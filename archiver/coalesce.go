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
	newbuf.add(readings)

	// check again and commit whatever buffer is there and put
	// our new buffer in. STILL NOT RIGHT
	c.Lock()
	if buf, found := c.buffers[uuid]; found {
		go c.commit(buf) //TODO: if goroutine, shouldn't return error
	}
	c.buffers[uuid] = newbuf
	c.Unlock()
	return err
}

// commits the buffer to the timeseries database
//TODO: this should send the error to some log. if an error happens
// during commiting, it should probably be fatal/critical, or should at least re-try
func (c *coalescer) commit(buf *streamBuffer) error {
	return c.tsStore.AddBuffer(buf)
}

// creates a new stream buffer and starts a goroutine to monitor
func (c *coalescer) getStreamBuffer(uuid UUID) *streamBuffer {
	return nil
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

/*
Start with:
RLOCK
1. is there a buffer already?
    yes:
        is it full?
            no: add and return RUNLOCK
            yes: continue to 2
    no:
        continue to 2
RUNLOCK
2. create a new buffer and put our readings in it
LOCK
3. is there a buffer already in the map (from step 1)?
    yes: it is full! we need to comimt! it is full if "found" is true and we are here
        go(?) commit the buffer
        LOCK
        put our new buffer  in the map
        UNLOCK
    no:
        we are the new buffer



there isn't a buffer, so we create one and put our readings in it
3. if (found) is true, then that means the buffer WAS full and needs to be replaced,
   so we commit it and then put our new buffer in the map. DONE





*/
