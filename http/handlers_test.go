package http

import (
	"fmt"
	"github.com/drewolson/testflight"
	"github.com/gtfierro/giles2/archiver"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testArchiver *archiver.Archiver

/*
Tests to run:

/add/:key
- no readings
- 1 reading
- lotsa readings
- missing uuid
- malformed json
- super large input

/api/query/:key

/api/query

*/

func TestHandleAdd(t *testing.T) {
	aConfig := archiver.LoadConfig("../giles.cfg")
	testArchiver = archiver.NewArchiver(aConfig)
	h := NewHTTPHandler(testArchiver)
	uuid := archiver.NewUUID()

	for _, test := range []struct {
		title          string
		toPost         string
		expectedStatus int
		expectedBody   string
	}{
		{
			"No Readings",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v" }}`, uuid),
			200,
			"",
		},
		{
			"Empty Readings 1",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Readings": [ ]}}`, uuid),
			200,
			"",
		},
		{
			"Empty Readings 2",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Readings": [[ ]]}}`, uuid),
			200,
			"",
		},
		{
			"Bad JSON 1",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Readings": [ ]]}}`, uuid),
			400,
			"invalid character ']' after object key:value pair",
		},
		{
			"Bad JSON 2",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Readings": [[ ]]}`, uuid),
			400,
			"unexpected EOF",
		},
	} {

		testflight.WithServer(h.handler, func(r *testflight.Requester) {
			response := r.Post("/add/dummykey", testflight.JSON, test.toPost)
			assert.Equal(t, test.expectedStatus, response.StatusCode, test.title)
			assert.Equal(t, test.expectedBody, response.Body, test.title)
		})
	}
}
