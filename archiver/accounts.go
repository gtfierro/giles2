package archiver

import (
	"crypto/hmac"
	"crypto/sha256"
	"gopkg.in/mgo.v2"
)

//TODO: put this in the config file
var secretkey = "abdef"

// these are the groups that users belong to
type Role struct {
	Name string
}

type User struct {
	Email    string
	Password []byte
	Roles    []Role
}

func newUser(email, password string) *User {
	//TODO: email validation?
	mac := hmac.New(sha256.New, []byte(secretkey+email))
	mac.Write([]byte(password))
	passwordMAC := mac.Sum(nil)
	//TODO: check if user already exists
	return &User{
		Email:    email,
		Password: passwordMAC,
		Roles:    []Role{},
	}
}

// add the given Role to user. Returns true if the user already
// had the role, and false otherwise. This method should always succeed
func (u *User) AddRole(role Role) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	// if we didn't find it, then append to the end
	u.Roles = append(u.Roles, role)
	return false
}

// this interface for managing user accounts should be implemented over some database
type AccountManager interface {
	// creates a new user if one does not already exist with the given
	// email, returns a pointer to that user and saves it to the database
	CreateUser(email, password string) (*User, error)
	// fetches/verifies a user and returns a pointer
	GetUser(email, password string) (*User, error)
	// Creates a new role with the given name and saves it to the database.
	// If a role already exists with this name, it will just return that role
	CreateRole(name string) (Role, error)
	// Removes the given role and strikes it from the role permissons of all streams
	// If the role does not exist, this is a noop
	RemoveRole(role Role) error
}

type mongoAccountManager struct {
	session *mgo.Session
	db      *mgo.Database
	users   *mgo.Collection
}

func newMongoAccountManager(c *mongoConfig) *mongoAccountManager {
	var err error
	ma := &mongoAccountManager{}
	ma.session, err = mgo.Dial(c.address.String())
	log.Notice("Connecting to MongoDB at %v...", c.address.String())
	if err != nil {
		log.Critical("Could not connect to MongoDB: %v", err)
		return nil
	}
	log.Notice("...connected!")
	// fetch/create collections and db reference
	ma.db = ma.session.DB("gilesauth")
	ma.users = ma.db.C("users")

	// add indexes. This will fail Fatal
	ma.addIndexes()
	return ma
}

func (ma *mongoAccountManager) addIndexes() {
	var err error
	// create indexes
	index := mgo.Index{
		Key:        []string{"uuid"},
		Unique:     true,
		DropDups:   false,
		Background: false,
		Sparse:     false,
	}
	err = ma.users.EnsureIndex(index)
	if err != nil {
		log.Fatalf("Could not create index on users.uuid (%v)", err)
	}
}

func (ma *mongoAccountManager) CreateUser(email, password string) (*User, error) {
	return nil, nil
}

func (ma *mongoAccountManager) GetUser(email, password string) (*User, error) {
	return nil, nil
}

func (ma *mongoAccountManager) CreateRole(name string) (Role, error) {
	return Role{}, nil
}

func (ma *mongoAccountManager) RemoveRole(role Role) error {
	return nil
}
