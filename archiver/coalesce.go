package archiver

import (
	"sync"
)

// thresholds for when transactions will be committed to the database
const (
	COALESCE_TIMEOUT = 1000  // milliseconds
	COALESCE_MAX     = 16384 // num readings
)

type coalescer struct {
	buffers map[UUID](*streamBuffer)
	tsStore TimeseriesStore
	sync.RWMutex
}

func newCoalescer(tsStore TimeseriesStore) *coalescer {
	return &coalescer{buffers: make(map[UUID](*streamBuffer)),
		tsStore: tsStore,
	}
}

// fetches a pre-existing or creates a new streamBuffer object which will
// buffer readings for a particular UUID (stream) and adds the readings
// from the given SmapMessage to it. When the threshold is hit,
// that is COALESCE_TIMEOUT milliseconds elapse or COALESCE_MAX readings are
// buffered, the buffer is committed to the timeseries database.
func (c *coalescer) add(uuid UUID, readings []*SmapNumberReading) error {
	var (
		buf    *streamBuffer
		newbuf *streamBuffer
		found  bool
		err    error
	)
	// grab a read lock to see if a buffer already exists
	c.RLock()
	if buf, found = c.buffers[uuid]; found {
		if buf.add(readings) { // returns true if copied into buffer
			c.RUnlock()
			return nil
		}
	}
	c.RUnlock()

	// create a new streambuffer
	newbuf = newStreamBuffer(uuid)

	// copy readings into it. Shouldn't need to check return value here
	// because it's a new buffer
	newbuf.add(readings)

	c.Lock()
	// after getting lock, check to see if a buffer has been created since
	// we gained the lock.
	if buf, found = c.buffers[uuid]; !found {
		// If there isn't, then we put our own buffer in place
		c.buffers[uuid] = newbuf
	} else {
		// write to the existing buffer
		//TODO: this is slow to do inside the lock section!
		//TODO: what if this is full too?
		buf.add(readings)
	}
	c.Unlock()
	return err
}

// commits the buffer to the timeseries database
func (c *coalescer) commit(uuid UUID) error {
	// lock the transaction coalescer, remove the buffer entry for use
	var (
		buf   *streamBuffer
		found bool
	)

	c.Lock()
	if buf, found = c.buffers[uuid]; !found {
		// if not found, just be a no-op
		c.Unlock()
		return nil
	}
	delete(c.buffers, uuid)
	c.Unlock()

	return c.tsStore.AddBuffer(buf)
}

type streamBuffer struct {
	readings []*SmapNumberReading
	idx      int
	uuid     UUID
	sync.RWMutex
}

func newStreamBuffer(uuid UUID) *streamBuffer {
	return &streamBuffer{readings: []*SmapNumberReading{},
		idx:  0,
		uuid: uuid}
}

// copy the readings into the buffer to be committed. Returns true if the
// copy was successful, and false otherwise.
func (sb *streamBuffer) add(readings []*SmapNumberReading) bool {

	// grab read lock and test that we aren't already full
	sb.RLock()
	//TODO: have a read lock for a timeout?
	if len(sb.readings) >= COALESCE_MAX {
		sb.RUnlock()
		return false
	}
	sb.RUnlock()

	// grab write lock and append readings to the buffer
	sb.Lock()
	sb.readings = append(sb.readings, readings...)
	sb.Unlock()
	return true
}
