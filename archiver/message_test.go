package archiver

import (
	"bytes"
	"encoding/json"
	"github.com/gtfierro/giles2/common"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"testing"
)

func Benchmarkcommon.SmapMessageToBsonBare(b *testing.B) {
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.ToBson()
	}
}

func Benchmarkcommon.SmapMessageToBsonFull(b *testing.B) {
	msg := &common.SmapMessage{
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

func Testcommon.SmapMessageToBson(t *testing.T) {
	myUUID := NewUUID()
	myPath := "/sensor8"

	for _, test := range []struct {
		msg *common.SmapMessage
		out bson.M
	}{
		{
			&common.SmapMessage{Path: myPath, UUID: myUUID},
			bson.M{"uuid": myUUID, "Path": myPath},
		},
		{
			&common.SmapMessage{Path: myPath, UUID: myUUID, Metadata: Dict{"System": "HVAC", "Point|Name": "My Point"}},
			bson.M{"uuid": myUUID, "Path": myPath, "Metadata.System": "HVAC", "Metadata.Point|Name": "My Point"},
		},
		{
			&common.SmapMessage{Path: myPath, UUID: myUUID, Actuator: Dict{"State": "[45, 90]"}},
			bson.M{"uuid": myUUID, "Path": myPath, "Actuator.State": "[45, 90]"},
		},
		{
			&common.SmapMessage{Path: myPath, UUID: myUUID, Properties: &SmapProperties{UnitOfTime: UOT_NS, UnitOfMeasure: "F", StreamType: NUMERIC_STREAM}},
			bson.M{"uuid": myUUID, "Path": myPath, "Properties.UnitofTime": UOT_NS, "Properties.UnitofMeasure": "F", "Properties.StreamType": NUMERIC_STREAM},
		},
	} {
		try := test.msg.ToBson()
		if !reflect.DeepEqual(try, test.out) {
			t.Errorf("common.SmapMessage should be \n%v\n but was \n%v\n", test.out, try)
		}
	}
}

func Testcommon.SmapMessageHasMetadata(t *testing.T) {
	myPath := "/sensor8"
	for _, test := range []struct {
		msg *common.SmapMessage
		out bool
	}{
		{&common.SmapMessage{Path: myPath, UUID: NewUUID()}, false},
		{&common.SmapMessage{Path: myPath, UUID: NewUUID(), Readings: []Reading{}}, false},
		{&common.SmapMessage{Path: myPath, UUID: NewUUID(), Metadata: Dict{}}, false},
		{&common.SmapMessage{Path: myPath, UUID: NewUUID(), Metadata: Dict{"X": "Y"}}, true},
		{&common.SmapMessage{Path: myPath, UUID: NewUUID(), Actuator: Dict{"X": "Y"}}, true},
		{&common.SmapMessage{Path: myPath, UUID: NewUUID(), Properties: &SmapProperties{}}, false},
		{&common.SmapMessage{Path: myPath, UUID: NewUUID(), Properties: &SmapProperties{UnitOfTime: UOT_NS}}, true},
	} {
		try := test.msg.HasMetadata()
		if try != test.out {
			t.Errorf("common.SmapMessage \n%v\n should be %v but was %v\n", test.msg, test.out, try)
		}
	}
}

func Testcommon.SmapMessageFromBson(t *testing.T) {
	myPath := "/sensor8"
	myUUID := NewUUID()
	myUUIDstr := string(myUUID)
	for _, test := range []struct {
		in  bson.M
		out *common.SmapMessage
	}{
		{
			bson.M{"uuid": myUUIDstr, "Path": myPath},
			&common.SmapMessage{UUID: myUUID, Path: myPath},
		},
		{
			bson.M{"uuid": myUUIDstr, "Path": myPath, "Metadata": bson.M{"System": "HVAC", "Point|Name": "Hey"}},
			&common.SmapMessage{UUID: myUUID, Path: myPath, Metadata: Dict{"System": "HVAC", "Point|Name": "Hey"}},
		},
		{
			bson.M{"uuid": myUUIDstr, "Path": myPath, "Properties": bson.M{"UnitofTime": UOT_NS, "UnitofMeasure": "F", "StreamType": NUMERIC_STREAM}},
			&common.SmapMessage{UUID: myUUID, Path: myPath, Properties: &SmapProperties{UnitOfTime: UOT_NS, UnitOfMeasure: "F", StreamType: NUMERIC_STREAM}},
		},
	} {
		try := common.SmapMessageFromBson(test.in)
		if !reflect.DeepEqual(*try, *test.out) {
			t.Errorf("common.SmapMessage \n%v\n should be \n%v\n but was \n%v\n", test.in, test.out, try)
		}
	}
}

func Benchmarkcommon.SmapMessageFromBson(b *testing.B) {
	in := bson.M{"uuid": string(NewUUID()), "Path": "/sensor8", "Metadata": bson.M{"System": "HVAC", "Point|Name": "Hey"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		common.SmapMessageFromBson(in)
	}
}

func Benchmarkcommon.SmapMessageDecodeJSON(b *testing.B) {
	var jsonstring = []byte(`{
    "/fast/sensor0": {
        "Readings": [[9182731928374, 30]],
        "uuid": "b86df176-6b40-5d58-8f29-3b85f5cfbf1e"
        }
    }`)
	var tsm Tieredcommon.SmapMessage
	var buf bytes.Buffer
	dec := json.NewDecoder(&buf)
	for i := 0; i < b.N; i++ {
		buf.Write(jsonstring)
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		dec.Decode(&tsm)
	}
}
