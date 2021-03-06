package archiver

import (
	"flag"
	"github.com/gtfierro/giles2/common"
	"gopkg.in/mgo.v2/bson"
	"net"
	"os"
	"reflect"
	"testing"
)

var ms *mongoStore
var testArchiver *Archiver

func TestMain(m *testing.M) {
	addr, err := net.ResolveTCPAddr("tcp4", "0.0.0.0:27017")
	if err != nil {
		log.Fatalf("%v", err)
	}
	config := &mongoConfig{
		address:     addr,
		enforceKeys: false,
	}
	ms = newMongoStore(config)
	pm = newMongoPermissionsManager(config)
	aConfig := LoadConfig("../giles.cfg")
	testArchiver = NewArchiver(aConfig)
	flag.Parse()
	os.Exit(m.Run())
}

func BenchmarkSaveTagsBare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		msg := &common.SmapMessage{
			Path: "/sensor8",
			UUID: common.NewUUID(),
		}
		ms.SaveTags(msg)
	}
}

func BenchmarkSaveTagsBareParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg := &common.SmapMessage{
				Path: "/sensor8",
				UUID: common.NewUUID(),
			}
			ms.SaveTags(msg)
		}
	})
}

func BenchmarkSaveTagsWithMetadata(b *testing.B) {
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Metadata: common.Dict{
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
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Metadata: common.Dict{
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
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Properties: &common.SmapProperties{
			UnitOfTime:    common.UOT_MS,
			StreamType:    common.NUMERIC_STREAM,
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
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Properties: &common.SmapProperties{
			UnitOfTime:    common.UOT_MS,
			StreamType:    common.NUMERIC_STREAM,
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
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Properties: &common.SmapProperties{
			UnitOfTime:    common.UOT_MS,
			StreamType:    common.NUMERIC_STREAM,
			UnitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)

	uot, err := ms.GetUnitOfTime(msg.UUID)
	if err != nil {
		t.Errorf("Err getting uot for %v (%v)", msg, err)
	}
	if uot != common.UOT_MS {
		t.Errorf("UOT should be %v but was %v", common.UOT_MS, uot)
	}
}

func TestGetStreamType(t *testing.T) {
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Properties: &common.SmapProperties{
			UnitOfTime:    common.UOT_MS,
			StreamType:    common.NUMERIC_STREAM,
			UnitOfMeasure: "F",
		},
	}
	ms.SaveTags(msg)

	st, err := ms.GetStreamType(msg.UUID)
	if err != nil {
		t.Errorf("Err getting StreamType for %v (%v)", msg, err)
	}
	if st != common.NUMERIC_STREAM {
		t.Errorf("UOT should be %v but was %v", common.NUMERIC_STREAM, st)
	}
}

func TestGetUnitOfMeasure(t *testing.T) {
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Properties: &common.SmapProperties{
			UnitOfTime:    common.UOT_MS,
			StreamType:    common.NUMERIC_STREAM,
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
	myUUID := common.NewUUID()
	myPath := "/sensor8"
	for _, test := range []struct {
		msg    *common.SmapMessage
		tags   []string
		where  bson.M
		result common.SmapMessageList
	}{
		{
			&common.SmapMessage{Path: myPath, UUID: myUUID},
			[]string{"uuid"},
			bson.M{"uuid": myUUID},
			common.SmapMessageList{{UUID: myUUID}},
		},
		{
			&common.SmapMessage{Path: myPath, UUID: myUUID, Metadata: common.Dict{"System": "HVAC", "Point|Name": "My Point"}},
			[]string{"Metadata.System", "Path"},
			bson.M{"uuid": myUUID},
			common.SmapMessageList{{Path: myPath, Metadata: common.Dict{"System": "HVAC"}}},
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
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Metadata: common.Dict{
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
	msg := &common.SmapMessage{
		Path: "/sensor8",
		UUID: common.NewUUID(),
		Metadata: common.Dict{
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
	commonUUID := common.NewUUID()
	msg1 := &common.SmapMessage{Path: "/sensor1", UUID: common.NewUUID(), Metadata: common.Dict{"Tag": "Value1", "Shared": string(commonUUID)}}
	msg2 := &common.SmapMessage{Path: "/sensor2", UUID: common.NewUUID(), Metadata: common.Dict{"Tag": "Value2", "Shared": string(commonUUID)}}
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
		for i := 0; i < len(test.result); i++ {
			if !reflect.DeepEqual(res[i], test.result[i]) {
				t.Errorf("Result should be \n%v\n but was \n%v\n", test.result[i], res[i])
			}
		}
	}
}

func TestGetUUIDs(t *testing.T) {
	commonUUID := common.NewUUID()
	msg1 := &common.SmapMessage{Path: "/sensor1", UUID: common.NewUUID(), Metadata: common.Dict{"Tag": "Value1", "Shared": string(commonUUID)}}
	msg2 := &common.SmapMessage{Path: "/sensor2", UUID: common.NewUUID(), Metadata: common.Dict{"Tag": "Value2", "Shared": string(commonUUID)}}
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
		t.Errorf("Results were %v but should be %v", results, []common.UUID{msg1.UUID, msg2.UUID})

	}
}
