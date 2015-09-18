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
	b.ReportAllocs()
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
			"Point|Name": "My Point",
		},
	}
	b.ReportAllocs()
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
			&SmapMessage{Path: myPath, UUID: myUUID, Metadata: Dict{"System": "HVAC", "Point|Name": "My Point"}},
			bson.M{"uuid": myUUID, "Path": myPath, "Metadata.System": "HVAC", "Metadata.Point|Name": "My Point"},
		},
		{
			&SmapMessage{Path: myPath, UUID: myUUID, Actuator: Dict{"State": "[45, 90]"}},
			bson.M{"uuid": myUUID, "Path": myPath, "Actuator.State": "[45, 90]"},
		},
		{
			&SmapMessage{Path: myPath, UUID: myUUID, Properties: smapProperties{UnitOfTime: UOT_NS, UnitOfMeasure: "F", StreamType: NUMERIC_STREAM}},
			bson.M{"uuid": myUUID, "Path": myPath, "Properties.UnitofTime": UOT_NS, "Properties.UnitofMeasure": "F", "Properties.StreamType": NUMERIC_STREAM},
		},
	} {
		try := test.msg.ToBson()
		if !reflect.DeepEqual(try, test.out) {
			t.Errorf("SmapMessage should be \n%v\n but was \n%v\n", test.out, try)
		}
	}
}

func TestSmapMessageHasMetadata(t *testing.T) {
	myPath := "/sensor8"
	for _, test := range []struct {
		msg *SmapMessage
		out bool
	}{
		{&SmapMessage{Path: myPath, UUID: NewUUID()}, false},
		{&SmapMessage{Path: myPath, UUID: NewUUID(), Readings: []Reading{}}, false},
		{&SmapMessage{Path: myPath, UUID: NewUUID(), Metadata: Dict{}}, false},
		{&SmapMessage{Path: myPath, UUID: NewUUID(), Metadata: Dict{"X": "Y"}}, true},
		{&SmapMessage{Path: myPath, UUID: NewUUID(), Actuator: Dict{"X": "Y"}}, true},
		{&SmapMessage{Path: myPath, UUID: NewUUID(), Properties: smapProperties{}}, false},
		{&SmapMessage{Path: myPath, UUID: NewUUID(), Properties: smapProperties{UnitOfTime: UOT_NS}}, true},
	} {
		try := test.msg.HasMetadata()
		if try != test.out {
			t.Errorf("SmapMessage \n%v\n should be %v but was %v\n", test.msg, test.out, try)
		}
	}
}

func TestSmapMessageFromBson(t *testing.T) {
	myPath := "/sensor8"
	myUUID := NewUUID()
	myUUIDstr := string(myUUID)
	for _, test := range []struct {
		in  bson.M
		out *SmapMessage
	}{
		{
			bson.M{"uuid": myUUIDstr, "Path": myPath},
			&SmapMessage{UUID: myUUID, Path: myPath},
		},
		{
			bson.M{"uuid": myUUIDstr, "Path": myPath, "Metadata": bson.M{"System": "HVAC", "Point|Name": "Hey"}},
			&SmapMessage{UUID: myUUID, Path: myPath, Metadata: Dict{"System": "HVAC", "Point|Name": "Hey"}},
		},
		{
			bson.M{"uuid": myUUIDstr, "Path": myPath, "Properties": bson.M{"UnitofTime": UOT_NS, "UnitofMeasure": "F", "StreamType": NUMERIC_STREAM}},
			&SmapMessage{UUID: myUUID, Path: myPath, Properties: smapProperties{UnitOfTime: UOT_NS, UnitOfMeasure: "F", StreamType: NUMERIC_STREAM}},
		},
	} {
		try := SmapMessageFromBson(test.in)
		if !reflect.DeepEqual(*try, *test.out) {
			t.Errorf("SmapMessage \n%v\n should be \n%v\n but was \n%v\n", test.in, test.out, try)
		}
	}
}

func BenchmarkSmapMessageFromBson(b *testing.B) {
	in := bson.M{"uuid": string(NewUUID()), "Path": "/sensor8", "Metadata": bson.M{"System": "HVAC", "Point|Name": "Hey"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SmapMessageFromBson(in)
	}
}
