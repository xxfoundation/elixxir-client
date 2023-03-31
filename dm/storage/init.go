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
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/dm"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

// MessageReceivedCallback is called any time a message is received or updated.
//
// messageUpdate is true if the Message already exists and was edited.
// conversationUpdate is true if the Conversation was created or modified.
type MessageReceivedCallback func(
	uuid uint64, pubKey ed25519.PublicKey, messageUpdate, conversationUpdate bool)

// impl implements the channels.EventModel interface with an underlying DB.
type impl struct {
	db                *gorm.DB // Stored database connection
	cipher            cryptoChannel.Cipher
	receivedMessageCB MessageReceivedCallback
}

// NewEventModel initializes the [channels.EventModel] interface with appropriate backend.
func NewEventModel(dbFilePath string, encryption cryptoChannel.Cipher,
	msgCb MessageReceivedCallback) (dm.EventModel, error) {
	model, err := newImpl(dbFilePath, encryption, msgCb)
	return dm.EventModel(model), err
}

func newImpl(dbFilePath string, encryption cryptoChannel.Cipher,
	msgCb MessageReceivedCallback) (*impl, error) {

	// Use a temporary, in-memory database if no path is specified
	if len(dbFilePath) == 0 {
		dbFilePath = temporaryDbPath
		jww.WARN.Printf("No database file path specified! " +
			"Using temporary in-memory database")
	}

	// Create the database connection
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
	err = db.AutoMigrate(&Conversation{}, &Message{})
	if err != nil {
		return nil, err
	}

	// Build the interface
	di := &impl{
		db:                db,
		cipher:            encryption,
		receivedMessageCB: msgCb,
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return di, nil
}
