////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/id"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/cmixproto"
	"gitlab.com/privategrity/client/switchboard"
	"fmt"
	"sync"
	"gitlab.com/privategrity/client/payment"
)

// Copy of the storage interface.
// It is identical to the interface used in Globals,
// and a results the types can be passed freely between the two
type Storage interface {
	// Give a Location for storage.  Does not need to be implemented if unused.
	SetLocation(string) error
	// Returns the Location for storage.
	// Does not need to be implemented if unused.
	GetLocation() string
	// Stores the passed byte slice
	Save([]byte) error
	// Returns the stored byte slice
	Load() []byte
}

//Message used for binding
type Message interface {
	// Returns the message's sender ID
	GetSender() []byte
	// Returns the message payload
	// Parse this with protobuf/whatever according to the type of the message
	GetPayload() []byte
	// Returns the message's recipient ID
	GetRecipient() []byte
	// Returns the message's type
	GetType() int32
}

//  Translate a bindings message to a parse message
// An object implementing this interface can be called back when the client
// gets a message of the type that the registerer specified at registration
// time.
type Listener interface {
	Hear(msg Message, isHeardElsewhere bool)
}

// Returns listener handle as a string.
// You can use it to delete the listener later.
// Please ensure userId has the correct length (256 bits)
// User IDs are informally big endian. If you want compatibility with the demo
// user names, set the last byte and leave all other bytes zero for userId.
// If you pass the zero user ID (256 bits of zeroes) to Listen() you will hear
// messages sent from all users.
// If you pass the zero type (just zero) to Listen() you will hear messages of
// all types.
func Listen(userId []byte, messageType int32, newListener Listener) string {
	typedUserId := new(id.UserID).SetBytes(userId)

	listener := &listenerProxy{proxy: newListener}

	return api.Listen(typedUserId, cmixproto.Type(messageType), listener, switchboard.Listeners)
}

func ListenToWallet(userId []byte, messageType int32, newListener Listener) string {
	typedUserId := new(id.UserID).SetBytes(userId)

	listener := &listenerProxy{proxy: newListener}

	return api.Listen(typedUserId, cmixproto.Type(messageType), listener,
		api.Wallet().Switchboard)
}

func FormatTextMessage(message string) []byte {
	return api.FormatTextMessage(message)
}

// Initializes the client by registering a storage mechanism and a reception
// callback.
// For the mobile interface, one must be provided
// The loc can be empty, it is only necessary if the passed storage interface
// requires it to be passed via "SetLocation"
//
// Parameters: storage implements Storage.
// Implement this interface to store the user session data locally.
// You must give us something for this parameter.
//
// loc is a string. If you're using DefaultStorage for your storage,
// this would be the filename of the file that you're storing the user
// session in.
func InitClient(storage Storage, loc string) error {
	if storage == nil {
		return errors.New("could not init client: Storage was nil")
	}

	proxy := &storageProxy{boundStorage: storage}
	err := api.InitClient(globals.Storage(proxy), loc)

	return err
}

// Registers user and returns the User ID.  Returns null if registration fails.
// registrationCode is a one time use string.
// nick is a nickname which must be 32 characters or less.
// nodeAddr is the ip address and port of the last node in the form: 192.168.1.1:50000
// numNodes is the number of nodes in the system
// Valid codes:
// 1
// “David”
// RUHPS2MI
// 2
// “Jim”
// AXJ3XIBD
// 3
// “Ben”
// AW55QN6U
// 4
// “Rick”
// XYRAUUO6
// 5
// “Spencer”
// UAV6IWD6
// 6
// “Jake”
// XEHCZT5U
// 7
// “Mario”
// BW7NEXOZ
// 8
// “Will”
// IRZVJ55Y
// 9
// “Allan”
// YRZEM7BW
// 10
// “Jono”
// OIF3OJ5I
func Register(registrationCode string, gwAddr string, numNodes int,
	mint bool) ([]byte, error) {

	if numNodes < 1 {
		return id.ZeroID[:], errors.New("invalid number of nodes")
	}

	UID, err := api.Register(registrationCode, gwAddr, uint(numNodes), mint)

	if err != nil {
		return id.ZeroID[:], err
	}

	return UID[:], nil
}

// Logs in the user based on User ID and returns the nickname of that user.
// Returns an empty string and an error
// UID is a uint64 BigEndian serialized into a byte slice
// TODO Pass the session in a proto struct/interface in the bindings or something
func Login(UID []byte, addr string) (string, error) {
	userID := new(id.UserID).SetBytes(UID)
	session, err := api.Login(userID, addr)
	return session.GetCurrentUser().Nick, err
}

//Sends a message structured via the message interface
// Automatically serializes the message type before the rest of the payload
// Returns an error if either sender or recipient are too short
func Send(m Message) error {
	sender := new(id.UserID).SetBytes(m.GetSender())
	recipient := new(id.UserID).SetBytes(m.GetRecipient())

	return api.Send(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type(m.GetType()),
			Body: m.GetPayload(),
		},
		Sender:   sender,
		Receiver: recipient,
	})
}

// Logs the user out, saving the state for the system and clearing all data
// from RAM
func Logout() error {
	return api.Logout()
}

// Returns the currently available balance in the wallet
func GetAvailableFunds() int64 {
	return int64(api.Wallet().GetAvailableFunds())
}

// Payer: user ID, 256 bits
// Value: must be positive
// Send the returned message unless you get an error
func Invoice(payer []byte, value int64, memo string) (Message, error) {
	userId := new(id.UserID).SetBytes(payer)
	msg, err := api.Wallet().Invoice(userId, value, memo)
	return &parse.BindingsMessageProxy{Proxy: msg}, err
}

// Get an invoice handle by listening to the wallet's PAYMENT_INVOICE_UI
// messages
// Returns a payment message that the bindings user can send at any time
// they wish
func Pay(invoiceHandle []byte) (Message, error) {
	var typedInvoiceId parse.MessageHash
	copiedLen := copy(typedInvoiceId[:], invoiceHandle)
	if copiedLen != parse.MessageHashLen {
		return nil, errors.New(fmt.Sprintf("Invoice ID wasn't long enough. " +
			"Got %v bytes, needed %v bytes.", copiedLen, parse.MessageHashLen))
	}

	msg, err := api.Wallet().Pay(typedInvoiceId)
	if err != nil {
		return nil, err
	}
	proxyMsg := parse.BindingsMessageProxy{Proxy: msg}
	return &proxyMsg, nil
	// After receiving a response from the payment bot,
	// the wallet will automatically send a receipt to the original sender of
	// the invoice. Make sure to listen for PAYMENT_RECEIPT_UI messages to
	// be able to handle the receipt.
}

// Gets a transaction from a specific transaction list in the wallet
// The UI should use this to display transaction details
// Use the proto buf enum to pass the transaction list to look up the handle in
// Use TransactionListTag in the protobuf to populate ListTag
// Transaction ID is used to look up the transaction in the list,
// and it must be 256 bits long.
// Will add a function to get all the transaction IDs in a transaction list
// later.
func GetTransaction(listTag int32, transactionID []byte) (*Transaction, error) {
	var transaction payment.Transaction
	var ok bool
	var messageId parse.MessageHash
	copiedLen := copy(messageId[:], transactionID)
	if copiedLen != parse.MessageHashLen {
		return nil, errors.New("The input transaction ID wasn't long enough")
	}
	transactionListTag := cmixproto.TransactionListTag(listTag)
	switch transactionListTag {
	case cmixproto.TransactionListTag_INBOUND_REQUESTS:
		transaction, ok = api.Wallet().GetInboundRequest(messageId)
	case cmixproto.TransactionListTag_OUTBOUND_REQUESTS:
		transaction, ok = api.Wallet().GetOutboundRequest(messageId)
	case cmixproto.TransactionListTag_PENDING_TRANSACTIONS:
		transaction, ok = api.Wallet().GetPendingTransaction(messageId)
	case cmixproto.TransactionListTag_COMPLETED_INBOUND_PAYMENTS:
		transaction, ok = api.Wallet().GetCompletedInboundPayment(messageId)
	case cmixproto.TransactionListTag_COMPLETED_OUTBOUND_PAYMENTS:
		transaction, ok = api.Wallet().GetCompletedOutboundPayment(messageId)
	}
	if !ok {
		return nil, fmt.Errorf("Couldn't find the transaction with id %q" +
			" in the list %v", transactionID, transactionListTag.String())
	} else {
		bindingsTransaction := Transaction{
			Memo:       transaction.Memo,
			Timestamp:  transaction.Timestamp.Unix(),
			Value:      int64(transaction.Value),
			InvoiceID:  transaction.OriginID[:],
			SenderID:   transaction.Sender[:],
			ReceiverID: transaction.Recipient[:],
		}
		return &bindingsTransaction, nil
	}
}

// Use TransactionListTag in the proto file to populate listTag
// Use TransactionListOrder in the proto file to get the transactions in the
// order you want
// The returned byte slice should really be a [][]byte instead. Each consecutive
// 256 bits or 32 bytes in the slice represents a different transaction ID that
// was in the list when this method was called.
func GetTransactionListIDs(listTag int32, order int32) []byte {
	transactionListTag := cmixproto.TransactionListTag(listTag)
	transactionListOrder := cmixproto.TransactionListOrder(order)
	return api.Wallet().GetTransactionIDs(transactionListTag, transactionListOrder)
}

// Meant to be read-only in the current version of the API
type Transaction struct {
	// What the transaction is for
	// e.g. "hardware", "coffee getting"
	Memo string
	// Time the transaction originated (UTC seconds since Unix epoch)
	Timestamp int64
	// Number of tokens transferred
	Value int64
	// ID of the invoice that initiated this transaction
	// TODO It would be nice to be able to look up these transactions just by
	// the ID of their originating invoice without having to do a linear search
	// or a sort
	// 256 bits long
	InvoiceID []byte
	// ID of the user who sends the tokens in the transaction
	// 256 bits long
	SenderID []byte
	// ID of the user who receives the tokens in the transaction
	// 256 bits long
	ReceiverID []byte
}

// Turns off blocking transmission so multiple messages can be sent
// simultaneously
func DisableBlockingTransmission() {
	api.DisableBlockingTransmission()
}

// Sets the minimum amount of time, in ms, between message transmissions
// Just for testing, probably to be removed in production
func SetRateLimiting(limit int) {
	api.SetRateLimiting(uint32(limit))
}

func RegisterForUserDiscovery(emailAddress string) error {
	return api.RegisterForUserDiscovery(emailAddress)
}

// FIXME This method doesn't get bound because of the exotic type it uses.
// Map types can't go over the boundary.
// The correct way to do over the boundary is to define
// a struct with a user ID and public key in it and return a
// pointer to that.
// Search() in bots only returns one user ID anyway. Returning a map would only
// be useful if a search could return more than one user.
func SearchForUser(emailAddress string) (map[uint64][]byte, error) {
	return api.SearchForUser(emailAddress)
}

// Translate a bindings listener to a switchboard listener
// Note to users of this package from other languages: Symbols that start with
// lowercase are unexported from the package and meant for internal use only.
type listenerProxy struct {
	proxy Listener
}

func (lp *listenerProxy) Hear(msg *parse.Message, isHeardElsewhere bool) {
	msgInterface := &parse.BindingsMessageProxy{Proxy: msg}
	lp.proxy.Hear(msgInterface, isHeardElsewhere)
}

// Unexported: Used to implement Lock and Unlock with the storage interface.
// Not quite sure whether this will work as intended or not. Will have to test.
type storageProxy struct {
	boundStorage Storage
	lock sync.Mutex
}

// TODO Should these methods take the mutex? Probably
func (s *storageProxy) SetLocation(location string) error {
	return s.boundStorage.SetLocation(location)
}

func (s *storageProxy) GetLocation() string {
	return s.boundStorage.GetLocation()
}

func (s *storageProxy) Save(data []byte) error {
	return s.boundStorage.Save(data)
}

func (s *storageProxy) Load() []byte {
	return s.boundStorage.Load()
}

func (s *storageProxy) Lock() {
	s.lock.Lock()
}

func (s *storageProxy) Unlock() {
	s.lock.Unlock()
}
