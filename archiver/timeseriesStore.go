package archiver

type TimeseriesStore interface {
	AddMessage(msg *SmapMessage) error

	// add the following SmapReading to the timeseries database
	AddBuffer(*streamBuffer) error

	// list of UUIDs, reference time, units of reference time
	// Retrieves data before the reference time for the given streams.
	//TODO: what is the return type here?
	Prev([]UUID, uint64, UnitOfTime) ([]SmapNumbersResponse, error)

	// list of UUIDs, reference time, units of reference time
	// Retrieves data after the reference time for the given streams.
	//TODO: what is the return type here?
	Next([]UUID, uint64, UnitOfTime) ([]SmapNumbersResponse, error)

	// uuids, start time, end time, unit of time
	GetData([]UUID, uint64, uint64, UnitOfTime) ([]SmapNumbersResponse, error)
}
