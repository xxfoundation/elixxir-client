////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"time"
)

// Message defines the SQL representation of a single Message.
//
// A Message belongs to one Channel.
//
// A Message may (informally) belong to one Message (Parent).
//
// The user's nickname can change each message, but the rest does not. We
// still duplicate all of it for each entry to simplify code for now.
type Message struct {
	Id              int64         `gorm:"primaryKey;autoIncrement:true"`
	Nickname        string        `gorm:"not null"`
	MessageId       []byte        `gorm:"uniqueIndex;not null"`
	ChannelId       []byte        `gorm:"index;not null"`
	ParentMessageId []byte        `gorm:"index"`
	Timestamp       time.Time     `gorm:"index;not null"`
	Lease           time.Duration `gorm:"not null"`
	Status          uint8         `gorm:"not null"`
	Text            []byte        `gorm:"not null"`
	Type            uint16        `gorm:"not null"`
	Round           int64         `gorm:"not null"`

	// Pointer to enforce zero-value reading in ORM.
	Hidden *bool `gorm:"not null"`
	Pinned *bool `gorm:"index;not null"`

	// User cryptographic Identity struct -- could be pulled out
	Pubkey         []byte `gorm:"not null"`
	DmToken        uint32 `gorm:"not null"`
	CodesetVersion uint8  `gorm:"not null"`
}

// Channel defines the SQL representation of a single Channel.
//
// A Channel has many Message.
type Channel struct {
	Id          []byte `gorm:"primaryKey;not null;autoIncrement:false"`
	Name        string `gorm:"not null"`
	Description string `gorm:"not null"`

	Messages []Message `gorm:"constraint:OnDelete:CASCADE"`
}

// File defines the SQL representation of a single File.
type File struct {
	// Id is a unique identifier for a given File.
	Id []byte `gorm:"primaryKey;not null;autoIncrement:false"`

	// Data stores the actual contents of the File.
	Data []byte

	// Link contains all the information needed to download the file data.
	Link []byte

	// Timestamp is the last time the file data, link, or status was modified.
	Timestamp time.Time `gorm:"not null"`

	// Status of the file in the event model.
	Status uint8 `gorm:"not null"`
}
