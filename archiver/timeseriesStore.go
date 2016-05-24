package archiver

import (
	"github.com/gtfierro/giles2/common"
)

type TimeseriesStore interface {
	AddMessage(msg *common.SmapMessage) error

	// list of UUIDs, reference time in nanoseconds
	// Retrieves data before the reference time for the given streams.
	//TODO: what is the return type here?
	Prev([]common.UUID, uint64) ([]common.SmapNumbersResponse, error)

	// list of UUIDs, reference time in nanoseconds
	// Retrieves data after the reference time for the given streams.
	//TODO: what is the return type here?
	Next([]common.UUID, uint64) ([]common.SmapNumbersResponse, error)

	// uuids, start time, end time (both in nanoseconds)
	GetData([]common.UUID, uint64, uint64) ([]common.SmapNumbersResponse, error)

	// returns true if the timestamp can be represented in the database
	ValidTimestamp(uint64, common.UnitOfTime) bool
}
