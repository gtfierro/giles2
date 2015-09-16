package archiver

import (
	"reflect"
	"testing"
)

var am AccountManager

func TestMongoCreateUser(t *testing.T) {
	if am.DeleteUser("gabe@example.com") != nil {
		t.Errorf("Could not delete old test user record")
		return
	}
	for _, test := range []struct {
		email    string
		pass     string
		hasError bool
	}{
		{"gabe@example.com", "12345", false},
		{"gabe@example.com", "12345", true},
		{"", "12345", true},
		{"gabe@example.com", "", true},
		{"", "", true},
	} {
		u, err := am.CreateUser(test.email, test.pass)
		if (err != nil) != test.hasError {
			t.Errorf("Expected error? %v. Got error? %v", test.hasError, err)
			continue
		}
		if test.hasError {
			continue
		}
		if u == nil {
			t.Errorf("Expected user, but user was nil")
			continue
		}
		if u.Email != test.email {
			t.Errorf("Generated user did not match given email: %v vs %v", u.Email, test.email)
			continue
		}
	}

	//cleanup
	if am.DeleteUser("gabe@example.com") != nil {
		t.Errorf("Could not delete old test user record")
	}
}

func TestMongoGetUser(t *testing.T) {
	email := "gabe@example.com"
	pass := "12345"
	//cleanup
	if am.DeleteUser(email) != nil {
		t.Errorf("Could not delete old test user record")
	}
	_, err := am.CreateUser(email, pass)
	if err != nil {
		t.Errorf("Could not create user %v %v %v", email, pass, err)
		return
	}

	for _, test := range []struct {
		email     string
		pass      string
		getErr    bool
		foundUser bool
	}{
		{"gabe@example.com", "12345", false, true},
		{"bad@example.com", "12345", true, false},
		{"gabe@example.com", "bad", false, false},
		{"", "", true, false},
	} {
		u, err := am.GetUser(test.email, test.pass)
		if (err != nil) != test.getErr {
			t.Errorf("Expected error? %v. Got error? %v", test.getErr, err)
			continue
		}
		if test.foundUser && u == nil {
			t.Errorf("Expected to find user but user is nil (%v)", test)
			continue
		} else if !test.foundUser && u != nil {
			t.Errorf("User was non-nil but should be nil", test)
			continue
		} else if !test.foundUser && u == nil {
			continue
		}

		if u.Email != test.email {
			t.Errorf("Generated user did not match given email: %v vs %v", u.Email, test.email)
			continue
		}
	}

	//cleanup
	if am.DeleteUser("gabe@example.com") != nil {
		t.Errorf("Could not delete old test user record")
	}
}

func TestMongoCreateRole(t *testing.T) {
	for _, test := range []struct {
		name  string
		found bool
	}{
		{"asdf", false},
		{"asdf", true},
	} {
		role, exists, err := am.CreateRole(test.name)
		if err != nil {
			t.Errorf("Error when creating role (%v) %v", test, err)
			continue
		}
		if exists != test.found {
			t.Errorf("Expected Role to exist? %v Did it? %v", test.found, exists)
			continue
		}
		if role.Name != test.name {
			t.Errorf("Returned role did not have the same name (%v)", test)
		}
	}
	err := am.RemoveRole("asdf")
	if err != nil {
		t.Errorf("Could not remove role %v", err)
	}
}

//TODO: finish these
func TestMongoUserAddGetRole(t *testing.T) {
	email := "gabe@example.com"
	pass := "12345"
	u, err := am.CreateUser(email, pass)
	if err != nil {
		t.Errorf("Could not create user record %v", err)
	}

	for _, test := range []struct {
		role     role
		roles    roleList
		hasError bool
	}{
		{role{"a"}, roleList{{"a"}}, false},
		{role{"a"}, roleList{{"a"}}, false},
		{role{"b"}, roleList{{"a"}, {"b"}}, false},
	} {
		err = am.UserAddRole(u, test.role)
		if (err != nil) != test.hasError {
			t.Errorf("Expected err? %v Got err? %v", test.hasError, err)
			continue
		}
		roles, err := am.UserGetRoles(u)
		if err != nil {
			t.Errorf("Could not fetch roles %v", err)
			continue
		}
		if !reflect.DeepEqual(roles, test.roles) {
			t.Errorf("Fetched roles were not equivalent. Has %v Expected %v", roles, test.roles)
			continue
		}
	}

	//cleanup
	if am.DeleteUser("gabe@example.com") != nil {
		t.Errorf("Could not delete old test user record")
	}
}

func TestMongoUserRemoveRole(t *testing.T) {}
func TestMongoUserGetRoles(t *testing.T)   {}
func TestMongoRemoveRole(t *testing.T) {
	email := "gabe@example.com"
	pass := "12345"
	u, err := am.CreateUser(email, pass)
	if err != nil {
		t.Errorf("Could not create user record %v", err)
	}
	for _, r := range []string{"a", "b", "c"} {
		err := am.UserAddRole(u, role{r})
		if err != nil {
			t.Errorf("Could not add role to user (%v)", err)
			return
		}
	}

	for _, test := range []struct {
		toDelete role
		roles    roleList
		hasError bool
	}{
		{role{"a"}, roleList{{"b"}, {"c"}}, false},
		{role{"a"}, roleList{{"b"}, {"c"}}, false},
		{role{"b"}, roleList{{"c"}}, false},
		{role{"c"}, roleList{}, false},
	} {
		err := am.UserRemoveRole(u, test.toDelete)
		if (err != nil) != test.hasError {
			t.Errorf("Expected err? %v Got err? %v", test.hasError, err)
			continue
		}
		roles, err := am.UserGetRoles(u)
		if err != nil {
			t.Errorf("Could not fetch roles %v", err)
			continue
		}
		if !reflect.DeepEqual(roles, test.roles) {
			t.Errorf("Fetched roles were not equivalent. Has %v Expected %v", roles, test.roles)
			continue
		}
	}
	//cleanup
	if am.DeleteUser("gabe@example.com") != nil {
		t.Errorf("Could not delete old test user record")
	}
}

func TestUserAddRole(t *testing.T) {
	u := &user{}
	for _, test := range []struct {
		toAdd     role
		goalRoles roleList
		duplicate bool
	}{
		{
			role{"a"},
			roleList{{"a"}},
			false,
		},
		{
			role{"b"},
			roleList{{"a"}, {"b"}},
			false,
		},
		{
			role{"a"},
			roleList{{"a"}, {"b"}},
			true,
		},
	} {
		duplicate := u.addRole(test.toAdd)
		if duplicate != test.duplicate {
			t.Errorf("Should role have been duplicate? %v Was it? %v", test.duplicate, duplicate)
			continue
		}

		if !reflect.DeepEqual(test.goalRoles, u.Roles) {
			t.Errorf("Role sets did not match: goal %v, user %v", test.goalRoles, u.Roles)
			continue
		}
	}
}

func TestUserRemoveRole(t *testing.T) {
	u := &user{Roles: roleList{{"a"}, {"b"}, {"c"}}}
	for _, test := range []struct {
		toRemove  role
		goalRoles roleList
		found     bool
	}{
		{
			role{"d"},
			roleList{{"a"}, {"b"}, {"c"}},
			false,
		},
		{
			role{"b"},
			roleList{{"a"}, {"c"}},
			true,
		},
		{
			role{"b"},
			roleList{{"a"}, {"c"}},
			false,
		},
		{
			role{"a"},
			roleList{{"c"}},
			true,
		},
		{
			role{"c"},
			roleList{},
			true,
		},
	} {
		found := u.removeRole(test.toRemove)
		if found != test.found {
			t.Errorf("Should role have been found? %v Was it? %v", test.found, found)
			continue
		}

		if !reflect.DeepEqual(test.goalRoles, u.Roles) {
			t.Errorf("Role sets did not match: goal %v, user %v", test.goalRoles, u.Roles)
			continue
		}
	}
}
