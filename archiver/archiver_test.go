package archiver

import (
	"github.com/gtfierro/giles2/common"
	"testing"
	"time"
)

func BenchmarkArchiverAddNoMetadata(b *testing.B) {
	msg := &common.SmapMessage{
		UUID: common.NewUUID(),
		Properties: &common.SmapProperties{
			UnitOfTime:    common.UOT_NS,
			UnitOfMeasure: "unit",
			StreamType:    common.NUMERIC_STREAM,
		},
		Readings: make([]common.Reading, 1),
	}
	ek := common.NewEphemeralKey()
	rdg := &common.SmapNumberReading{Time: 0, Value: 0}
	msg.Readings[0] = rdg
	offset := time.Now().UnixNano()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rdg.Time = uint64(offset + int64(i))
		msg.Readings[0] = rdg
		testArchiver.AddData(msg, ek)
		msg.Properties = nil
	}

}
