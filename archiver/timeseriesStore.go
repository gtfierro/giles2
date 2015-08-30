package archiver

type TimeseriesStore interface {
	// add the following SmapMessage to the timeseries database
	Add(msg *SmapMessage) error

	// list of UUIDs, reference time, units of reference time
	// Retrieves data before the reference time for the given streams.
	//TODO: what is the return type here?
	Prev([]UUID, uint64, UnitOfTime) ([]SmapNumbersResponse, error)
}
