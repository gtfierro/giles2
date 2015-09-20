package archiver

type TimeseriesStore interface {
	AddMessage(msg *SmapMessage) error

	// add the following SmapReading to the timeseries database
	AddBuffer(*streamBuffer) error

	// list of UUIDs, reference time in nanoseconds
	// Retrieves data before the reference time for the given streams.
	//TODO: what is the return type here?
	Prev([]UUID, uint64) ([]SmapNumbersResponse, error)

	// list of UUIDs, reference time in nanoseconds
	// Retrieves data after the reference time for the given streams.
	//TODO: what is the return type here?
	Next([]UUID, uint64) ([]SmapNumbersResponse, error)

	// uuids, start time, end time (both in nanoseconds)
	GetData([]UUID, uint64, uint64) ([]SmapNumbersResponse, error)
}
