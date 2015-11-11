package archiver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/vmihailenco/msgpack.v2"
	"io/ioutil"
	"testing"
)

var smallMessage = SmapMessage{
	UUID: "d24325e6-1d7d-11e2-ad69-a7c2fa8dba61",
	Readings: []Reading{
		&SmapNumberReading{1351043674000, 4.4},
		&SmapNumberReading{1351043675000, 5.0},
	},
}

var smallMessage2 = SmapMessage2{
	UUID: "d24325e6-1d7d-11e2-ad69-a7c2fa8dba61",
	Readings: []*SmapNumberReading{
		&SmapNumberReading{1351043674000, 4.4},
		&SmapNumberReading{1351043675000, 5.0},
	},
}

var bigMessage = SmapMessage{
	UUID: "d24325e6-1d7d-11e2-ad69-a7c2fa8dba61",
	Properties: &SmapProperties{
		UnitOfTime:    UOT_MS,
		UnitOfMeasure: "Watt",
		StreamType:    NUMERIC_STREAM,
	},
	Metadata: Dict{
		"Location/Room":     "410",
		"Location/Building": "XYZ",
		"Point/Type":        "Sensor",
		"Point/Measure":     "Power",
	},
	Readings: []Reading{
		&SmapNumberReading{1351043674000, 4.4},
		&SmapNumberReading{1351043675000, 5.0},
	},
}

var bigMessage2 = SmapMessage2{
	UUID: "d24325e6-1d7d-11e2-ad69-a7c2fa8dba61",
	Properties: &SmapProperties2{
		UnitOfTime:    uint64(UOT_MS),
		UnitOfMeasure: "Watt",
		StreamType:    uint64(NUMERIC_STREAM),
	},
	Metadata: Dict{
		"Location/Room":     "410",
		"Location/Building": "XYZ",
		"Point/Type":        "Sensor",
		"Point/Measure":     "Power",
	},
	Readings: []*SmapNumberReading{
		&SmapNumberReading{1351043674000, 4.4},
		&SmapNumberReading{1351043675000, 5.0},
	},
}

func BenchmarkPaperDecodeJSONBig(b *testing.B) {
	var big = []byte(`{
      "Metadata": {
        "Location/Room": 410,
        "Location/Building": "XYZ",
        "Point/Type": "Sensor",
        "Point/Measure": "Power"
      },
      "Properties": {
        "Timezone": "America/Los_Angeles",
        "UnitofMeasure": "Watt",
        "UnitofTime": "ms",
        "StreamType": "numeric"
      },
      "Readings": [
        [1351043674000, 4.4],
        [1351043675000, 5.0]
      ],
      "uuid": "d24325e6-1d7d-11e2-ad69-a7c2fa8dba61"
    }`)
	var buf bytes.Buffer
	var ism SmapMessage
	dec := json.NewDecoder(&buf)
	for i := 0; i < b.N; i++ {
		buf.Write(big)
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		err := dec.Decode(&ism)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func BenchmarkPaperDecodeJSONSmall(b *testing.B) {
	var small = []byte(`{
      "Readings": [
        [1351043674000, 4.4],
        [1351043675000, 5.0]
      ],
      "uuid": "d24325e6-1d7d-11e2-ad69-a7c2fa8dba61"
    }`)
	var buf bytes.Buffer
	var ism SmapMessage
	dec := json.NewDecoder(&buf)
	for i := 0; i < b.N; i++ {
		buf.Write(small)
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		err := dec.Decode(&ism)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func BenchmarkPaperEncodeJSONBig(b *testing.B) {
	enc := json.NewEncoder(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		enc.Encode(bigMessage)
	}
	var buf bytes.Buffer
	enc = json.NewEncoder(&buf)
	enc.Encode(bigMessage)
	fmt.Println("big JSON:", buf.Len())
}

func BenchmarkPaperEncodeJSONSmall(b *testing.B) {
	enc := json.NewEncoder(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		enc.Encode(smallMessage2)
	}
	var buf bytes.Buffer
	enc = json.NewEncoder(&buf)
	enc.Encode(smallMessage2)
	fmt.Println("small JSON:", buf.Len())
}

func BenchmarkPaperDecodeMsgPackBig(b *testing.B) {
	var buf []byte
	buf, _ = msgpack.Marshal(&bigMessage2)
	var sm SmapMessage2
	for i := 0; i < b.N; i++ {
		msgpack.Unmarshal(buf, &sm)
	}
}

func BenchmarkPaperDecodeMsgPackSmall(b *testing.B) {
	var buf []byte
	buf, _ = msgpack.Marshal(&smallMessage2)
	var sm SmapMessage2
	for i := 0; i < b.N; i++ {
		msgpack.Unmarshal(buf, &sm)
	}
}

func BenchmarkPaperEncodeMsgPackBig(b *testing.B) {
	var buf []byte
	for i := 0; i < b.N; i++ {
		buf, _ = msgpack.Marshal(&bigMessage2)
	}
	fmt.Println("big msgpack", len(buf))
}

func BenchmarkPaperEncodeMsgPackSmall(b *testing.B) {
	var buf []byte
	for i := 0; i < b.N; i++ {
		buf, _ = msgpack.Marshal(&smallMessage2)
	}
	fmt.Println("small msgpack", len(buf))
}

func BenchmarkPaperEncodeCapnpBig(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bigMessage2.Save(&buf)
	}
	fmt.Println("big capnp", buf.Len())
}

func BenchmarkPaperDecodeCapnpBig(b *testing.B) {
	var buf bytes.Buffer
	bigMessage2.Save(&buf)
	var sm SmapMessage2
	for i := 0; i < b.N; i++ {
		sm.Load(&buf)
	}
}

func BenchmarkPaperEncodeCapnpSmall(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		smallMessage2.Save(&buf)
	}
	fmt.Println("small capnp", buf.Len())
}

func BenchmarkPaperDecodeCapnpSmall(b *testing.B) {
	var buf bytes.Buffer
	smallMessage2.Save(&buf)
	var sm SmapMessage2
	for i := 0; i < b.N; i++ {
		sm.Load(&buf)
	}
}
