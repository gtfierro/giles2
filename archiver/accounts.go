package archiver

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"sync/atomic"
)

// 1. Need to create Roles.
// 2. Then, need to be able to assign a user to a role. This should
// 	be stored with the user record, rather than with the role record.
// 3. Then, generate an ephemeral key for a user that has an expiry.
//	Need to think about how to then cache the key->Role->stream lookups
//  so they are as fast as possible
// Before this, we need to think about how we will be accessing that. When
// I receive a message of some sort, I will also receive an ephemeral key.
// For each stream that I need to access, I need to ask that stream
// if the provided ephemeral key has permission to do what it wants to do.

//TODO: put this in the config file
var secretkey = "abdef"

// these are the groups that users belong to
type role struct {
	Name string
}

type roleList []role

func (r roleList) GetBSON() (interface{}, error) {
	var s = make([]string, len(r))
	for i, r := range r {
		s[i] = r.Name
	}
	return s, nil
}

func (r *roleList) SetBSON(raw bson.Raw) error {
	var (
		s []string
		t map[string][]string
	)
	err := raw.Unmarshal(&t)
	if err != nil {
		return err
	}
	s = t["roles"]
	*r = make(roleList, len(s))
	for i, name := range s {
		(*r)[i] = role{name}
	}
	return nil
}

type user struct {
	Email    string
	Password []byte
	Roles    roleList
}

// add the given Role to user. Returns true if the user already
// had the role, and false otherwise. This method should always succeed
func (u *user) addRole(role role) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	// if we didn't find it, then append to the end
	u.Roles = append(u.Roles, role)
	return false
}

// remove the role from the user. Returns true if user had the role
// and false otherwise (it is okay to remove a role from a user
// even if the user doesn't have that role)
func (u *user) removeRole(role role) bool {
	for i, r := range u.Roles {
		if r == role {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			return true
		}
	}
	return false
}

// this interface for managing user accounts should be implemented over some database
type permissionsManager interface {
	// creates a new user if one does not already exist with the given
	// email, returns a pointer to that user and saves it to the database
	CreateUser(email, password string) (*user, error)
	// fetches/verifies a user and returns a pointer
	GetUser(email, password string) (*user, error)
	// removes the user with the given email
	DeleteUser(email string) error

	UserAddRole(*user, role) error
	UserRemoveRole(*user, role) error
	UserGetRoles(*user) (roleList, error)

	// Creates a new role with the given name and saves it to the database.
	// If a role already exists with this name, it will just return that role.
	// The boolean value is true if the Role already existed, an false otherwise
	CreateRole(name string) (role, bool, error)
	// Removes the given role and strikes it from the role permissons of all streams
	// If the role does not exist, this is a noop
	RemoveRole(name string) error

	//TODO: we do really want the ephemeral key cache to be common to the system.
	// maybe it is worth making permissions manager an actual struct
	// with backend-independent features, and then having the db-specific
	// part provide lower level functions

	// returns true if the given ephemeral key is valid
	ValidEphemeralKey(EphemeralKey) bool
	// generates a new ephemeral key for the given user
	NewEphemeralKey(*user) EphemeralKey
	// revokes an ephemeral key, either through timeout or administrative intervention
	RevokeEphemeralKey(EphemeralKey)
}

type mongoPermissionsManager struct {
	session     *mgo.Session
	db          *mgo.Database
	users       *mgo.Collection
	roles       *mgo.Collection
	ephKeyCache atomic.Value //map[EphemeralKey]struct{}
}

func newMongoPermissionsManager(c *mongoConfig) *mongoPermissionsManager {
	var err error
	ma := &mongoPermissionsManager{}
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
	ma.ephKeyCache.Store(make(map[EphemeralKey]struct{}))

	// add indexes. This will fail Fatal
	ma.addIndexes()
	return ma
}

func (ma *mongoPermissionsManager) addIndexes() {
	var err error
	// create indexes
	index := mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		DropDups:   false,
		Background: false,
		Sparse:     false,
	}
	err = ma.users.EnsureIndex(index)
	if err != nil {
		log.Fatalf("Could not create index on users.email (%v)", err)
	}

	index.Key[0] = "name"
	err = ma.roles.EnsureIndex(index)
	if err != nil {
		log.Fatalf("Could not create index on roles.name (%v)", err)
	}
}

// Creates a new user with the given email and password. Email should be unique.
// Method will error out if user already exists
func (ma *mongoPermissionsManager) CreateUser(email, password string) (u *user, err error) {
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
	u = &user{Email: email}
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
func (ma *mongoPermissionsManager) GetUser(email, password string) (u *user, err error) {
	q := ma.users.Find(bson.M{"email": email})

	// test if we have a user with that email
	num, err := q.Count()
	if err != nil {
		return
	} else if num == 0 {
		err = fmt.Errorf("No user with that email %v", email)
		return
	}

	u = &user{}

	// have reader, so extract into object
	err = q.One(u)
	if err != nil {
		return
	}

	// match password
	if !verifyPassword(u.Password, []byte(password)) {
		// password no match!
		u = nil
	}
	return
}

func (ma *mongoPermissionsManager) DeleteUser(email string) error {
	// we use RemoveAll instead of Remove because Remove returns
	// an error if document isn't found, and we don't care here
	_, err := ma.users.RemoveAll(bson.M{"email": email})
	//TODO: purge user from all caches
	return err
}

// check the db to see if a role with this name already exists. If it does, return it.
// if not, create and then return.
func (ma *mongoPermissionsManager) CreateRole(name string) (r role, exists bool, err error) {
	q := ma.roles.Find(bson.M{"name": name})
	exists = false

	// test if we have a matching role
	num, err := q.Count()
	if err != nil {
		return
	} else if num != 0 {
		exists = true
		q.One(&r)
		return
	}

	// here, we create a new role with that name
	r = role{name}
	err = ma.roles.Insert(r)
	return
}

// add the given role to the given user
//TODO: update caches of ephemeral keys associated w/ this user?
func (ma *mongoPermissionsManager) UserAddRole(u *user, r role) error {
	// if this returns true, then we already have the role
	if !u.addRole(r) {
		return ma.users.Update(bson.M{"email": u.Email}, u)
	}
	return nil
}

func (ma *mongoPermissionsManager) UserRemoveRole(u *user, r role) error {
	if u.removeRole(r) { // update happened
		return ma.users.Update(bson.M{"email": u.Email}, u)
	}
	// update didn't happen
	return nil
}

func (ma *mongoPermissionsManager) UserGetRoles(u *user) (roleList, error) {
	//TODO: how do we know if our user passed in is up to date?
	// assume user doesn't know its roles
	var roles roleList
	err := ma.users.Find(bson.M{"email": u.Email}).Select(bson.M{"roles": 1, "_id": 0}).One(&roles)
	for _, r := range roles {
		u.addRole(r)
	}
	return roles, err
}

// remove the role and remove mentions of it from all streams. This is a lengthy
// operation.
func (ma *mongoPermissionsManager) RemoveRole(name string) error {
	//TODO: remove role from all streams and from all users and caches
	_, err := ma.roles.RemoveAll(bson.M{"name": name})
	return err
}

func (ma *mongoPermissionsManager) ValidEphemeralKey(e EphemeralKey) bool {
	cache := ma.ephKeyCache.Load().(map[EphemeralKey]struct{})
	_, isValid := cache[e]
	return isValid
}

// TODO: implement this
func (ma *mongoPermissionsManager) NewEphemeralKey(u *user) EphemeralKey {
	return EphemeralKey("")
}

// TODO: implement this
func (ma *mongoPermissionsManager) RevokeEphemeralKey(e EphemeralKey) {
}

// generates a new password
func generatePasswordHash(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, 10)
}

// returns true if passwored matches hashed
func verifyPassword(hashed, password []byte) bool {
	return bcrypt.CompareHashAndPassword(hashed, password) == nil
}
