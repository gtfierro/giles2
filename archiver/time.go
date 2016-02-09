package archiver

import (
	"fmt"
	"strconv"
	"time"
)

func parseAbsTime(num, units string) (time.Time, error) {
	var d time.Time
	var err error
	i, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return d, err
	}
	uot, err := ParseUOT(units)
	if err != nil {
		return d, err
	}
	unixseconds, err := convertTime(i, uot, UOT_S)
	if err != nil {
		return d, err
	}
	tmp, err := convertTime(unixseconds, UOT_S, uot)
	if err != nil {
		return d, err
	}
	leftover := i - tmp
	unixns, err := convertTime(leftover, uot, UOT_NS)
	if err != nil {
		return d, err
	}
	d = time.Unix(int64(unixseconds), int64(unixns))
	return d, err
}

func parseReltime(num, units string) (time.Duration, error) {
	var d time.Duration
	i, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return d, err
	}
	d = time.Duration(i)
	switch units {
	case "h", "hr", "hour", "hours":
		d *= time.Hour
	case "m", "min", "minute", "minutes":
		d *= time.Minute
	case "s", "sec", "second", "seconds":
		d *= time.Second
	case "us", "usec", "microsecond", "microseconds":
		d *= time.Microsecond
	case "ms", "msec", "millisecond", "milliseconds":
		d *= time.Millisecond
	case "ns", "nsec", "nanosecond", "nanoseconds":
		d *= time.Nanosecond
	case "d", "day", "days":
		d *= 24 * time.Hour
	default:
		err = fmt.Errorf("Invalid unit %v. Must be h,m,s,us,ms,ns,d", units)
	}
	return d, err
}

func ParseUOT(units string) (UnitOfTime, error) {
	switch units {
	case "s", "sec", "second", "seconds":
		return UOT_S, nil
	case "us", "usec", "microsecond", "microseconds":
		return UOT_US, nil
	case "ms", "msec", "millisecond", "milliseconds":
		return UOT_MS, nil
	case "ns", "nsec", "nanosecond", "nanoseconds":
		return UOT_NS, nil
	default:
		return UOT_S, fmt.Errorf("Invalid unit %v. Must be s,us,ms,ns", units)
	}
}

// Takes 2 durations and returns the result of them added together
func addDurations(d1, d2 time.Duration) time.Duration {
	d1nano := d1.Nanoseconds()
	d2nano := d2.Nanoseconds()
	res := d1nano + d2nano
	return time.Duration(res) * time.Nanosecond
}
