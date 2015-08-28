package archiver

import (
	"flag"
	"net"
	"os"
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
	msg := &SmapMessage{
		Path: "/sensor8",
		UUID: NewUUID(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.SaveTags(msg)
	}
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
