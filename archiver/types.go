package archiver

import (
	"github.com/satori/go.uuid"
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
	OBJECT_STREAM StreamType = iota
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
