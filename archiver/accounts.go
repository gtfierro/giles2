package archiver

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 1. Need to create Roles.
// 2. Then, need to be able to assign a user to a role. This should
// 	be stored with the user record, rather than with the role record.
// 3. Then, generate an ephemeral key for a user that has an expiry.
//	Need to think about how to then cache the key->Role->stream lookups
//  so they are as fast as possible

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
	// removes the user with the given email
	DeleteUser(email string) error
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
	roles   *mgo.Collection
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
	ma.roles = ma.db.C("roles")

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

// Creates a new user with the given email and password. Email should be unique.
// Method will error out if user already exists
func (ma *mongoAccountManager) CreateUser(email, password string) (u *User, err error) {
	if len(email) == 0 || len(password) == 0 {
		err = fmt.Errorf("Email and password must be of length > 0")
		return
	}

	q := ma.users.Find(bson.M{"email": email})

	// test if we have a user with that email
	num, err := q.Count()
	if err != nil {
		return
	} else if num != 0 {
		err = fmt.Errorf("User already exists with email %v", email)
		return
	}

	// create a new user
	u = &User{Email: email}
	// encode password
	u.Password, err = generatePasswordHash([]byte(password))
	if err != nil {
		return
	}
	err = ma.users.Insert(u)
	return
}

// Fetches user with given email only if the password matches. Returns an error
// if user doesn't exist or if password does not match
func (ma *mongoAccountManager) GetUser(email, password string) (u *User, err error) {
	q := ma.users.Find(bson.M{"email": email})

	// test if we have a user with that email
	num, err := q.Count()
	if err != nil {
		return
	} else if num == 0 {
		err = fmt.Errorf("No user with that email %v", email)
		return
	}

	// have reader, so extract into object
	err = q.One(u)
	if err != nil {
		return
	}

	// match password
	if verifyPassword(u.Password, []byte(password)) {
		// password no match!
		u = nil
	}
	return
}

func (ma *mongoAccountManager) DeleteUser(email string) error {
	return ma.users.Remove(bson.M{"email": email})
}

// check the db to see if a role with this name already exists. If it does, return it.
// if not, create and then return.
func (ma *mongoAccountManager) CreateRole(name string) (r Role, err error) {
	q := ma.roles.Find(bson.M{"name": name})

	// test if we have a matching role
	num, err := q.Count()
	if err != nil {
		return
	} else if num != 0 {
		q.One(&r)
		return
	}

	// here, we create a new role with that name
	r = Role{name}
	err = ma.roles.Insert(r)
	return
}

func (ma *mongoAccountManager) RemoveRole(role Role) error {
	// remove the role and remove mentions of it from all streams. This is a lengthy
	// operation.
	return nil
}

// generates a new password
func generatePasswordHash(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, 10)
}

// returns true if passwored matches hashed
func verifyPassword(hashed, password []byte) bool {
	return bcrypt.CompareHashAndPassword(hashed, password) == nil
}