package http

import (
	"fmt"
	"github.com/drewolson/testflight"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var testArchiver *giles.Archiver

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
	aConfig := giles.LoadConfig("../giles.cfg")
	testArchiver = giles.NewArchiver(aConfig)
	h := NewHTTPHandler(testArchiver)
	uuid := common.NewUUID()

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
		{
			"Bad Readings 1: negative timestamp",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Readings": [[-1, 0]]}}`, uuid),
			400,
			"json: cannot unmarshal number -1 into Go value of type uint64",
		},
		{
			"Bad Readings 2: too big timestamp",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Readings": [[1000000000000000000, 0]]}}`, uuid),
			500,
			"Bad Timestamp: 1000000000000000000",
		},
		{
			"Bad Readings 3: too big timestamp",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Readings": [[3458764513820540929, 0]]}}`, uuid),
			500,
			"Bad Timestamp: 3458764513820540929",
		},
		{
			"Good readings: 1 reading max timestamp",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Properties": {"UnitofTime": "ns"}, "Readings": [[3458764513820540928, 0]]}}`, uuid),
			200,
			"",
		},
		{
			"Good readings: 2 repeat readings",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Properties": {"UnitofTime": "ns"}, "Readings": [[1000000000000000000, 0], [1000000000000000000, 0]]}}`, uuid),
			200,
			"",
		},
		{
			"Good readings: 2 repeat readings diff values",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Properties": {"UnitofTime": "ns"}, "Readings": [[2000000000000000000, 0], [2000000000000000000, 1]]}}`, uuid),
			200,
			"",
		},
		{
			"Lots of readings",
			fmt.Sprintf(`{"/sensor/0": {"Path": "/sensor/0", "uuid": "%v", "Properties": {"UnitofTime": "ns"}, "Readings": [ %v [1000000000000000000, 0]]}}`, uuid, strings.Repeat("[1000000000000000000, 0],", 1000)),
			200,
			"",
		},
	} {

		testflight.WithServer(h.handler, func(r *testflight.Requester) {
			response := r.Post("/add/dummykey", testflight.JSON, test.toPost)
			assert.Equal(t, test.expectedStatus, response.StatusCode, test.title)
			assert.Equal(t, test.expectedBody, response.Body, test.title)
		})
	}
}
