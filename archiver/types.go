package archiver

import (
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"
	"strings"
)

// internal unique identifier
type UUID string

// generates a new random v4 UUID
func NewUUID() UUID {
	return UUID(uuid.NewV4().String())
}

// a flat map for storing key-value pairs
type Dict map[string]string

func NewDict() *Dict {
	return new(Dict)
}

func DictFromBson(m bson.M) *Dict {
	d := Dict{}
	for k, v := range m {
		if vs, ok := v.(string); ok {
			d[k] = vs
		}
	}
	return &d
}

// unit of time indicators
type UnitOfTime uint

const (
	// nanoseconds 1000000000
	UOT_NS UnitOfTime = 1
	// microseconds 1000000
	UOT_US UnitOfTime = 2
	// milliseconds 1000
	UOT_MS UnitOfTime = 3
	// seconds 1
	UOT_S UnitOfTime = 4
)

var unitmultiplier = map[UnitOfTime]uint64{
	UOT_NS: 1000000000,
	UOT_US: 1000000,
	UOT_MS: 1000,
	UOT_S:  1,
}

// Takes a timestamp with accompanying unit of time 'stream_uot' and
// converts it to the unit of time 'target_uot'
func convertTime(time uint64, stream_uot, target_uot UnitOfTime) uint64 {
	if stream_uot == target_uot {
		return time
	}
	if target_uot < stream_uot { // target/stream is > 1, so we can use uint64
		return time * (unitmultiplier[target_uot] / unitmultiplier[stream_uot])
	} else {
		return time / uint64(unitmultiplier[stream_uot]/unitmultiplier[target_uot])
	}
}

func (u UnitOfTime) String() string {
	switch u {
	case UOT_NS:
		return "ns"
	case UOT_US:
		return "us"
	case UOT_MS:
		return "ms"
	case UOT_S:
		return "s"
	default:
		return ""
	}
}

func (u *UnitOfTime) UnmarshalJSON(b []byte) (err error) {
	str := strings.Trim(string(b), `"`)
	switch str {
	case "ns":
		*u = UOT_NS
	case "us":
		*u = UOT_US
	case "ms":
		*u = UOT_MS
	case "s":
		*u = UOT_S
	default:
		return fmt.Errorf("%v is not a valid UnitOfTime", str)
	}
	return nil
}

// stream type indicators
type StreamType uint

const (
	OBJECT_STREAM StreamType = iota + 1
	NUMERIC_STREAM
)

func (st StreamType) String() string {
	switch st {
	case OBJECT_STREAM:
		return "object"
	case NUMERIC_STREAM:
		return "numeric"
	default:
		return ""
	}
}

func (st *StreamType) UnmarshalJSON(b []byte) (err error) {
	str := strings.Trim(string(b), `"`)
	switch str {
	case "numeric":
		*st = NUMERIC_STREAM
	case "object":
		*st = OBJECT_STREAM
	default:
		return fmt.Errorf("%v is not a valid StreamType", str)
	}
	return nil
}

type ApiKey string

type EphemeralKey string

//TODO: we will attach validation/generation methods to this type
