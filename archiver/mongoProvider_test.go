package archiver

import (
	"flag"
	"gopkg.in/mgo.v2/bson"
	"net"
	"os"
	"reflect"
	"testing"
)

var ms *mongoStore

func TestMain(m *testing.M) {
	addr, err := net.ResolveTCPAddr("tcp4", "0.0.0.0:27017")
	if err != nil {
		log.Fatal("%v", err)
	}
	config := &mongoConfig{
		address:     addr,
		enforceKeys: false,
	}
	ms = newMongoStore(config)
	pm = newMongoPermissionsManager(config)
	flag.Parse()
	os.Exit(m.Run())
}

func BenchmarkSaveTagsBare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		msg := &SmapMessage{
			Path: "/sensor8",
			UUID: NewUUID(),
		}
		ms.SaveTags(msg)
	}
}

func BenchmarkSaveTagsBareParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg := &SmapMessage{
				Path: "/sensor8",
				UUID: NewUUID(),
			}
			ms.SaveTags(msg)
		}
	})
}

func BenchmarkSaveTagsWithMetadata(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Metadata: Dict{
			"System":     "HVAC",
			"Point.Name": "My Point",
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.SaveTags(msg)
	}
}

func BenchmarkSaveTagsWithMetadataParallel(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Metadata: Dict{
			"System":     "HVAC",
			"Point.Name": "My Point",
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ms.SaveTags(msg)
		}
	})
}

func BenchmarkGetUnitOfTime(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Properties: smapProperties{
			UnitOfTime:    UOT_MS,
			StreamType:    NUMERIC_STREAM,
			UnitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.GetUnitOfTime(msg.UUID)
	}
}

func BenchmarkGetUnitOfTimeParallel(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Properties: smapProperties{
			UnitOfTime:    UOT_MS,
			StreamType:    NUMERIC_STREAM,
			UnitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ms.GetUnitOfTime(msg.UUID)
		}
	})
}

func TestGetUnitOfTime(t *testing.T) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Properties: smapProperties{
			UnitOfTime:    UOT_MS,
			StreamType:    NUMERIC_STREAM,
			UnitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)

	uot, err := ms.GetUnitOfTime(msg.UUID)
	if err != nil {
		t.Errorf("Err getting uot for %v (%v)", msg, err)
	}
	if uot != UOT_MS {
		t.Errorf("UOT should be %v but was %v", UOT_MS, uot)
	}
}

func TestGetStreamType(t *testing.T) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Properties: smapProperties{
			UnitOfTime:    UOT_MS,
			StreamType:    NUMERIC_STREAM,
			UnitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)

	st, err := ms.GetStreamType(msg.UUID)
	if err != nil {
		t.Errorf("Err getting StreamType for %v (%v)", msg, err)
	}
	if st != NUMERIC_STREAM {
		t.Errorf("UOT should be %v but was %v", NUMERIC_STREAM, st)
	}
}

func TestGetUnitOfMeasure(t *testing.T) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Properties: smapProperties{
			UnitOfTime:    UOT_MS,
			StreamType:    NUMERIC_STREAM,
			UnitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)

	uom, err := ms.GetUnitOfMeasure(msg.UUID)
	if err != nil {
		t.Errorf("Err getting UnitofMeasure for %v (%v)", msg, err)
	}
	if uom != "F" {
		t.Errorf("UOT should be %v but was %v", "F", uom)
	}
}

func TestGetTags(t *testing.T) {
	myUUID := NewUUID()
	myPath := "/sensor8"
	for _, test := range []struct {
		msg    *SmapMessage
		tags   []string
		where  bson.M
		result *SmapMessageList
	}{
		{
			&SmapMessage{Path: myPath, UUID: myUUID},
			[]string{"uuid"},
			bson.M{"uuid": myUUID},
			&SmapMessageList{{UUID: myUUID}},
		},
		{
			&SmapMessage{Path: myPath, UUID: myUUID, Metadata: Dict{"System": "HVAC", "Point|Name": "My Point"}},
			[]string{"Metadata.System", "Path"},
			bson.M{"uuid": myUUID},
			&SmapMessageList{{Path: myPath, Metadata: Dict{"System": "HVAC"}}},
		},
	} {
		ms.SaveTags(test.msg)
		res, err := ms.GetTags(test.tags, test.where)
		if err != nil {
			t.Errorf("Err during GetTags (%v) \n%v", err, test)
		}
		if !reflect.DeepEqual(test.result, res) {
			t.Errorf("Result should be \n%v\n but was \n%v\n", test.result.ToBson(), res.ToBson())
		}
	}
}

func BenchmarkGetTags(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Metadata: Dict{
			"System":     "HVAC",
			"Point.Name": "My Point",
		},
	}
	tags := []string{"uuid"}
	where := bson.M{"uuid": msg.UUID}
	ms.SaveTags(msg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.GetTags(tags, where)
	}
}

func BenchmarkGetTagsParallel(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Metadata: Dict{
			"System":     "HVAC",
			"Point.Name": "My Point",
		},
	}
	tags := []string{"uuid"}
	where := bson.M{"uuid": msg.UUID}
	ms.SaveTags(msg)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ms.GetTags(tags, where)
		}
	})
}

func TestGetDistinct(t *testing.T) {
	commonUUID := NewUUID()
	msg1 := &SmapMessage{Path: "/sensor1", UUID: NewUUID(), Metadata: Dict{"Tag": "Value1", "Shared": string(commonUUID)}}
	msg2 := &SmapMessage{Path: "/sensor2", UUID: NewUUID(), Metadata: Dict{"Tag": "Value2", "Shared": string(commonUUID)}}
	ms.SaveTags(msg1)
	ms.SaveTags(msg2)
	for _, test := range []struct {
		tag    string
		where  bson.M
		result []string
	}{
		{
			"Metadata.Tag",
			bson.M{"Metadata.Shared": commonUUID},
			[]string{"Value1", "Value2"},
		},
	} {
		res, err := ms.GetDistinct(test.tag, test.where)
		if err != nil {
			t.Errorf("Err during GetDistinct (%v) \n%v", err, test)
		}
		if !compareStringSliceAsSet(res, test.result) {
			t.Errorf("Result should be \n%v\n but was \n%v\n", test.result, res)
		}
	}
}

func TestGetUUIDs(t *testing.T) {
	commonUUID := NewUUID()
	msg1 := &SmapMessage{Path: "/sensor1", UUID: NewUUID(), Metadata: Dict{"Tag": "Value1", "Shared": string(commonUUID)}}
	msg2 := &SmapMessage{Path: "/sensor2", UUID: NewUUID(), Metadata: Dict{"Tag": "Value2", "Shared": string(commonUUID)}}
	ms.SaveTags(msg1)
	ms.SaveTags(msg2)

	results, err := ms.GetUUIDs(bson.M{"Metadata.Shared": string(commonUUID)})

	if err != nil {
		t.Errorf("Error running GetUUIDs (%v)", err)
		return
	}

	if len(results) != 2 {
		t.Errorf("Should have returned 2, but returned %v", len(results))
		return
	}

	if !((results[0] == msg1.UUID || results[1] == msg1.UUID) &&
		(results[0] == msg2.UUID || results[1] == msg2.UUID)) {
		t.Errorf("Results were %v but should be %v", results, []UUID{msg1.UUID, msg2.UUID})

	}
}
