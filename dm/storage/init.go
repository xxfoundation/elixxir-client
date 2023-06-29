////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles low-level database control and interfaces.

package storage

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gitlab.com/elixxir/client/v4/dm"
	"gitlab.com/elixxir/crypto/message"
)

// MessageReceivedCallback is called any time a message is received or updated.
//
// messageUpdate is true if the Message already exists and was edited.
// conversationUpdate is true if the Conversation was created or modified.
type MessageReceivedCallback func(
	uuid uint64, pubKey ed25519.PublicKey, messageUpdate, conversationUpdate bool)

// Callbacks contains callbacks that are used by this event model implementation.
type Callbacks interface {
	// MessageReceived is called any time a message is received or updated.
	//
	// messageUpdate is true if the message already exists and was edited.
	// conversationUpdate is true if the conversation was created or modified.
	MessageReceived(uuid uint64, pubKey ed25519.PublicKey,
		messageUpdate, conversationUpdate bool)

	// MessageDeleted is called when a message is deleted.
	MessageDeleted(messageID message.ID)
}

// impl implements the dm.EventModel interface with an underlying DB.
// NOTE: This model is NOT thread safe - it is the responsibility of the
// caller to ensure that its methods are called sequentially.
type impl struct {
	db  *gorm.DB // Stored database connection
	cbs Callbacks
}

// NewEventModel initializes the [dm.EventModel] interface with appropriate backend.
func NewEventModel(dbFilePath string, cbs Callbacks) (dm.EventModel, error) {
	useTemporary := len(dbFilePath) == 0
	model, err := newImpl(dbFilePath, cbs, useTemporary)
	return dm.EventModel(model), err
}

// If useTemporary is set to true, this will use an in-RAM database.
func newImpl(dbFilePath string, cbs Callbacks, useTemporary bool) (*impl, error) {

	if useTemporary {
		dbFilePath = fmt.Sprintf(temporaryDbPath, dbFilePath)
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
		db:  db,
		cbs: cbs,
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return di, nil
}
