package archiver

import (
	"testing"
	"time"
)

func BenchmarkArchiverAddNoMetadata(b *testing.B) {
	msg := &SmapMessage{
		UUID: NewUUID(),
		Properties: &SmapProperties{
			UnitOfTime:    UOT_NS,
			UnitOfMeasure: "unit",
			StreamType:    NUMERIC_STREAM,
		},
		Readings: make([]Reading, 1),
	}
	ek := NewEphemeralKey()
	rdg := &SmapNumberReading{Time: 0, Value: 0}
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
