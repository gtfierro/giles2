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
		Properties: &smapProperties{
			unitOfTime:    UOT_MS,
			streamType:    NUMERIC_STREAM,
			unitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.GetUnitOfTime(msg.UUID)
	}
}

func BenchmarkGetUnitOfTimeParallel(b *testing.B) {
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
		Properties: &smapProperties{
			unitOfTime:    UOT_MS,
			streamType:    NUMERIC_STREAM,
			unitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)
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
		Properties: &smapProperties{
			unitOfTime:    UOT_MS,
			streamType:    NUMERIC_STREAM,
			unitOfMeasure: "F",
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
		Properties: &smapProperties{
			unitOfTime:    UOT_MS,
			streamType:    NUMERIC_STREAM,
			unitOfMeasure: "F",
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
		Properties: &smapProperties{
			unitOfTime:    UOT_MS,
			streamType:    NUMERIC_STREAM,
			unitOfMeasure: "F",
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
