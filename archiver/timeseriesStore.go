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
	GetData(uuids []common.UUID, start uint64, end uint64) ([]common.SmapNumbersResponse, error)

	// pointWidth is the log of the number of records to aggregate
	StatisticalData(uuids []common.UUID, pointWidth int, start, end uint64) ([]common.StatisticalNumbersResponse, error)

	// width in nanoseconds
	WindowData(uuids []common.UUID, width, start, end uint64) ([]common.StatisticalNumbersResponse, error)

	// delete data
	DeleteData(uuids []common.UUID, start uint64, end uint64) error

	// returns true if the timestamp can be represented in the database
	ValidTimestamp(uint64, common.UnitOfTime) bool
}
