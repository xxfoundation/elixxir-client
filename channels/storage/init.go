////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles low level database control and interfaces

package storage

import (
	"crypto/ed25519"
	"github.com/pkg/errors"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
)

// UiCallbacks contains a subset of the methods on [channels.UiCallbacks] that
// are used by this event model implementation.
type UiCallbacks interface {
	ChannelUpdate(channelID *id.ID, delete bool)
	MessageReceived(uuid int64, channelID *id.ID, update bool)
	UserMuted(channelID *id.ID, pubKey ed25519.PublicKey, unmute bool)
	MessageDeleted(messageID message.ID)
}

// impl implements the channels.EventModel interface with an underlying DB.
type impl struct {
	db  *gorm.DB // Stored database connection
	cbs UiCallbacks
}

// NewEventModel initializes the [channels.EventModel] interface with appropriate backend.
func NewEventModel(dbFilePath string, uiCallbacks UiCallbacks) (channels.EventModel, error) {
	model, err := newImpl(dbFilePath, uiCallbacks)
	return channels.EventModel(model), err
}

func newImpl(dbFilePath string, uiCallbacks UiCallbacks) (*impl, error) {

	// Use a temporary, in-memory database if no path is specified
	if len(dbFilePath) == 0 {
		dbFilePath = temporaryDbPath
		jww.WARN.Printf("No database file path specified! " +
			"Using temporary in-memory database")
	}

	// Create the database connection
	jww.INFO.Printf("Opening DB file at %s...", dbFilePath)
	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{
		Logger: logger.New(jww.TRACE, logger.Config{LogLevel: logger.Info}),
	})
	if err != nil {
		return nil, errors.Errorf("Unable to initialize database backend: %+v", err)
	}

	// Enable foreign keys because they are disabled in SQLite by default
	if err = db.Exec("PRAGMA foreign_keys = ON", nil).Error; err != nil {
		return nil, err
	}

	// Enable Write Ahead Logging to enable multiple DB connections
	if err = db.Exec("PRAGMA journal_mode = WAL;", nil).Error; err != nil {
		return nil, err
	}

	// Get and configure the internal database ConnPool
	sqlDb, err := db.DB()
	if err != nil {
		return nil, errors.Errorf(
			"Unable to configure database connection pool: %+v", err)
	}

	// TODO: Configure these options appropriately for mobile client. Maybe they should be configurable?
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDb.SetMaxIdleConns(5)
	// SetMaxOpenConns sets the maximum number of open connections to the Database.
	sqlDb.SetMaxOpenConns(10)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be idle.
	sqlDb.SetConnMaxIdleTime(5 * time.Minute)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDb.SetConnMaxLifetime(10 * time.Minute)

	// Initialize the database schema
	// WARNING: Order is important. Do not change without database testing
	err = db.AutoMigrate(&Channel{}, &Message{}, File{})
	if err != nil {
		return nil, err
	}

	// Build the interface
	di := &impl{
		db:  db,
		cbs: uiCallbacks,
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return di, nil
}
