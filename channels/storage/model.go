////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"time"
)

// Channel defines the database representation of a single Channel.
type Channel struct {
	Id          []byte `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string `gorm:"not null"`

	// A Channel has many Message.
	Messages []Message `gorm:"constraint:OnDelete:CASCADE"`
}

// Message defines the database representation of a single Message.
//
// A Message belongs to one Channel.
//
// A Message may (informally) belong to one Message (Parent).
type Message struct {
	Id              uint64        `gorm:"primaryKey;autoIncrement:true"`
	Nickname        string        `gorm:"not null"`
	MessageId       []byte        `gorm:"uniqueIndex;not null"`
	ChannelId       []byte        `gorm:"index;not null"`
	ParentMessageId []byte        `gorm:"index"`
	Timestamp       time.Time     `gorm:"index;not null"`
	Lease           time.Duration `gorm:"not null"`
	Status          uint8         `gorm:"not null"`
	Text            []byte        `gorm:"not null"`
	Type            uint16        `gorm:"not null"`
	Round           uint64        `gorm:"not null"`

	// Pointer to enforce zero-value reading in ORM.
	Hidden *bool `gorm:"not null"`
	Pinned *bool `gorm:"index;not null"`

	// User cryptographic Identity struct -- could be pulled out
	Pubkey         []byte `gorm:"not null"`
	DmToken        uint32 `gorm:"not null"`
	CodesetVersion uint8  `gorm:"not null"`

	// A Message may have one File.
	File *File `gorm:"constraint:OnDelete:CASCADE"`
}

// File defines the database representation of a single File.
//
// A File may belong to one Message.
type File struct {
	Id        uint64 `gorm:"primaryKey;autoIncrement:true"`
	MessageId []byte `gorm:"not null"`
	Data      []byte `gorm:"not null"`
}
