package globals

// Globally instantiated UserSession
var Session = newUserSession()

// Interface for User Session operations
type UserSession interface {
	Login(id uint64) (isValidUser bool)
	GetCurrentUser() (currentUser *User)
}

// Creates a new UserSession interface
func newUserSession() UserSession {
	// With an underlying Session data structure
	return UserSession(&sessionObj{currentUser: nil})
}

// Struct holding relevant session data
type sessionObj struct {
	// Currently authenticated user
	currentUser *User
}

// Set CurrentUser to the user corresponding to the given id
// if it exists. Return a bool for whether the given id exists
func (s *sessionObj) Login(id uint64) (isValidUser bool) {
	user, userExists := Users.GetUser(id)
	// User must exist and no User can be previously logged in
	if isValidUser = userExists && s.GetCurrentUser() == nil; isValidUser {
		s.currentUser = user
	}
	return
}

// Return a copy of the current user
func (s *sessionObj) GetCurrentUser() (currentUser *User) {
	if s.currentUser != nil {
		// Explicit deep copy
		currentUser = &User{
			Id: s.currentUser.Id,
			Nick:s.currentUser.Nick,
		}
	}
	return
}
