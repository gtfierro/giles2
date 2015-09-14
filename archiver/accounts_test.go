package archiver

import (
	"testing"
)

var am AccountManager

func TestCreateUser(t *testing.T) {
	if am.DeleteUser("gabe@example.com") != nil {
		t.Errorf("Could not delete old test user record")
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
			return
		}
		if test.hasError {
			return
		}
		if u == nil {
			t.Errorf("Expected user, but user was nil")
			return
		}
		if u.Email != test.email {
			t.Errorf("Generated user did not match given email: %v vs %v", u.Email, test.email)
			return
		}
	}
}

func TestGetUser(t *testing.T) {
}
