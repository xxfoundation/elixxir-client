package globals

// Globally instantiated UserSession
var Session = newUserSession()

// Interface for User Session operations
type UserSession interface {
	Login(id uint64) (isValidUser bool)
	GetCurrentUser() (currentUser User)
}

// Creates a new UserSession interface
func newUserSession() UserSession {
	// With an underlying Session data structure
	return UserSession(&sessionObj{})
}

// Struct holding relevant session data
type sessionObj struct {
	// Currently authenticated user
	currentUser *User
}

// Set CurrentUser to the user corresponding to the given id
// if it exists. Return a bool for whether the given id exists
func (s sessionObj) Login(id uint64) (isValidUser bool) {
	user, isValidUser := Users.GetUser(id)
	if isValidUser {
		s.currentUser = user
	}
	return
}

// Return a copy of the current user
func (s sessionObj) GetCurrentUser() (currentUser User) {
	currentUser = *s.currentUser
	return
}
