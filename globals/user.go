package globals

// Globally instantiated UserRegistry
var Users = newUserRegistry()

// Interface for User Registry operations
type UserRegistry interface {
	GetUser(id uint64) (user *User, ok bool)
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

	uc[0] = &User{Id: 1, Nick: "Phineas Flynn"}
	uc[1] = &User{Id: 2, Nick: "Ferb Flynn"}
	uc[2] = &User{Id: 3, Nick: "Cadance Flynn"}
	uc[3] = &User{Id: 4, Nick: "Perry the Platypus"}
	uc[4] = &User{Id: 5, Nick: "Heinz Doofenshmirtz"}

	// With an underlying UserMap data structure
	return UserRegistry(&UserMap{userCollection: uc, idCounter: 0})
}

// Struct representing a User in the system
type User struct {
	Id   uint64
	Nick string
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id uint64) (user *User, ok bool) {
	user, ok = m.userCollection[id]
	return
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}
