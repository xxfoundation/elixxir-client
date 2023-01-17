////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"time"
)

// Message defines the IndexedDb representation of a single Message.
//
// A Message belongs to one Channel.
//
// A Message may (informally) belong to one Message (Parent).
//
// The user's nickname can change each message, but the rest does not. We
// still duplicate all of it for each entry to simplify code for now.
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
}

// Channel defines the IndexedDb representation of a single Channel.
//
// A Channel has many Message.
type Channel struct {
	Id          []byte `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string `gorm:"not null"`

	Messages []Message `gorm:"constraint:OnDelete:CASCADE"`
}
