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
	mdStore MetadataStore
	sync.RWMutex
}

func newCoalescer(tsStore TimeseriesStore, mdStore MetadataStore) *coalescer {
	return &coalescer{buffers: make(map[UUID](*streamBuffer)),
		tsStore: tsStore,
		mdStore: mdStore,
	}
}

// fetches a pre-existing or creates a new streamBuffer object which will
// buffer readings for a particular UUID (stream) and adds the readings
// from the given SmapMessage to it. When the threshold is hit,
// that is COALESCE_TIMEOUT milliseconds elapse or COALESCE_MAX readings are
// buffered, the buffer is committed to the timeseries database.
func (c *coalescer) add(msg *SmapMessage) error {
	var (
		buf        *streamBuffer
		newbuf     *streamBuffer
		found      bool
		stream_uot UnitOfTime
		err        error
	)

	if len(msg.Readings) == 0 {
		return nil // no readings to commit
	}

	// if the message has properties, grab unit of time, else grab from cache
	// grab the unit of time for this stream
	if msg.Properties.UnitOfTime != 0 {
		stream_uot = msg.Properties.UnitOfTime
	} else if stream_uot, err = c.mdStore.GetUnitOfTime(msg.UUID); err != nil {
		return err
	}

	// convert readings to nanoseconds
	if stream_uot != UOT_NS {
		for _, rdg := range msg.Readings {
			rdg.ConvertTime(stream_uot, UOT_NS)
		}
	}

	// now we are ready to commit into a buffer
	c.RLock()
	buf, found = c.buffers[msg.UUID]
	c.RUnlock()

	if found && buf.add(msg.Readings) {
		c.Lock()
		//TODO: commit buf
		delete(c.buffers, msg.UUID)
		c.Unlock()
		return nil
	}
	c.RUnlock()

	// if no buffer exists, then we create our own
	newbuf = newStreamBuffer(msg.UUID)
	newbuf.Lock()
	newbuf.readings = append(newbuf.readings, msg.Readings...)
	newbuf.Unlock()

	// now we need to write this back to the internal map
	// grab the lock
	c.Lock()
	// check that a new buffer hasn't been allocated already
	if buf, found = c.buffers[msg.UUID]; found {
		if buf.add(msg.Readings) {
			//TODO: commit buf
			delete(c.buffers, msg.UUID)
		}
	} else {
		c.buffers[msg.UUID] = newbuf
	}
	c.Unlock()

	return nil
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
	readings []Reading
	idx      int
	uuid     UUID
	sync.RWMutex
}

func newStreamBuffer(uuid UUID) *streamBuffer {
	return &streamBuffer{readings: []Reading{},
		idx:  0,
		uuid: uuid}
}

// copy the readings into the buffer to be committed. Returns true if the
// buffer is ready to be deleted, false otherwise
func (sb *streamBuffer) add(readings []Reading) bool {
	// grab write lock and append readings to the buffer
	sb.Lock()
	sb.readings = append(sb.readings, readings...)
	sb.Unlock()

	// grab read lock and test that we aren't already full
	sb.RLock()
	//TODO: have a read lock for a timeout?
	if len(sb.readings) >= COALESCE_MAX {
		sb.RUnlock()
		return true
	}
	sb.RUnlock()
	return false
}
