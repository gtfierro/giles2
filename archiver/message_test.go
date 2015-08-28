package archiver

import (
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"testing"
)

func BenchmarkSmapMessageToBsonBare(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.ToBson()
	}
}

func BenchmarkSmapMessageToBsonFull(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Metadata: Dict{
			"System":     "HVAC",
			"Point.Name": "My Point",
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.ToBson()
	}
}

func TestSmapMessageToBson(t *testing.T) {
	myUUID := NewUUID()
	myPath := "/sensor8"

	for _, test := range []struct {
		msg *SmapMessage
		out bson.M
	}{
		{
			&SmapMessage{Path: myPath, UUID: myUUID},
			bson.M{"uuid": myUUID, "Path": myPath},
		},
		{
			&SmapMessage{Path: myPath, UUID: myUUID, Metadata: Dict{"System": "HVAC", "Point.Name": "My Point"}},
			bson.M{"uuid": myUUID, "Path": myPath, "Metadata.System": "HVAC", "Metadata.Point.Name": "My Point"},
		},
		{
			&SmapMessage{Path: myPath, UUID: myUUID, Actuator: Dict{"State": "[45, 90]"}},
			bson.M{"uuid": myUUID, "Path": myPath, "Actuator.State": "[45, 90]"},
		},
		{
			&SmapMessage{Path: myPath, UUID: myUUID, Properties: &smapProperties{UOT_NS, "F", NUMERIC_STREAM}},
			bson.M{"uuid": myUUID, "Path": myPath, "Properties.UnitofTime": "ns", "Properties.UnitofMeasure": "F", "Properties.StreamType": "numeric"},
		},
	} {
		try := test.msg.ToBson()
		if !reflect.DeepEqual(try, test.out) {
			t.Errorf("Smap Message should be \n%v\n but was \n%v\n", test.out, try)
		}
	}
}
