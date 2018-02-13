package globals

// Globally instantiated UserRegistry
var Users = newUserRegistry()

// Interface for User Registry operations
type UserRegistry interface {
	GetUser(id uint64) *User
	CountUsers() int
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
	// Increments sequentially for User.id values
	idCounter uint64
}

// Creates a new UserRegistry interface
func newUserRegistry() UserRegistry {

	uc := make(map[uint64]*User)

	uc[0] = &User{Id: 0, Nick: "Phineas Flynn"}
	uc[1] = &User{Id: 1, Nick: "Ferb Flynn"}
	uc[2] = &User{Id: 2, Nick: "Cadance Flynn"}
	uc[3] = &User{Id: 3, Nick: "Perry the Platypus"}
	uc[4] = &User{Id: 4, Nick: "Heinz Doofenshmirtz"}

	// With an underlying UserMap data structure
	return UserRegistry(&UserMap{userCollection: uc, idCounter: 0})
}

type User struct {
	Id   uint64
	Nick string
}

// GetUser returns a user with the given ID from userCollection.
func (m *UserMap) GetUser(id uint64) *User {
	// If key does not exist, return nil
	return m.userCollection[id]
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}
