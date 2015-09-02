package archiver

import (
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"
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

type ApiKey string

//TODO: we will attach validation/generation methods to this type
