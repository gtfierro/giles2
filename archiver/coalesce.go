package archiver

import (
	"sync"
	"time"
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
	bufpool sync.Pool
	sync.RWMutex
}

func newCoalescer(tsStore TimeseriesStore, mdStore MetadataStore) *coalescer {
	return &coalescer{buffers: make(map[UUID](*streamBuffer)),
		tsStore: tsStore,
		mdStore: mdStore,
		bufpool: sync.Pool{
			New: func() interface{} {
				return &streamBuffer{readings: make([]Reading, COALESCE_MAX), idx: 0, stop: nil}
			},
		},
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

	// go will evalute these until it finds a FALSE
	if found && buf.fits(msg.Readings) && buf.add(msg.Readings) {
		c.Lock()
		delete(c.buffers, msg.UUID)
		go c.commit(buf)
		c.Unlock()
		return nil
	} else if found && !buf.fits(msg.Readings) {
		c.Lock()
		delete(c.buffers, msg.UUID)
		go c.commit(buf)
		c.Unlock()
	}

	// if no buffer exists, then we create our own
	newbuf = c.newStreamBuffer(msg.UUID)
	newbuf.add(msg.Readings)

	// now we need to write this back to the internal map
	// grab the lock
	c.Lock()
	// check that a new buffer hasn't been allocated already
	if buf, found = c.buffers[msg.UUID]; found {
		if buf.fits(msg.Readings) {
			if buf.add(msg.Readings) {
				delete(c.buffers, msg.UUID)
				go c.commit(buf)
			}
		} else {
			delete(c.buffers, msg.UUID)
			go c.commit(buf)
		}
	} else {
		c.buffers[msg.UUID] = newbuf
	}
	c.Unlock()

	return nil
}

func (c *coalescer) newStreamBuffer(uuid UUID) *streamBuffer {
	//TODO: fetch this from a pool of streambuffers
	newbuf := c.bufpool.Get().(*streamBuffer)
	newbuf.uuid = uuid
	//newbuf.stop = time.AfterFunc(time.Duration(COALESCE_TIMEOUT)*time.Millisecond, func() {
	//	c.Lock()
	//	delete(c.buffers, uuid)
	//	c.Unlock()
	//	c.commit(newbuf)
	//})
	return newbuf
}

// commits the buffer to the timeseries database
//TODO: this should send the error to some log. if an error happens
// during commiting, it should probably be fatal/critical, or should at least re-try
func (c *coalescer) commit(buf *streamBuffer) error {
	if buf.stop != nil {
		buf.stop.Stop()
	}
	err := c.tsStore.AddBuffer(buf)
	buf.idx = 0
	buf.uuid = ""
	buf.stop = nil
	c.bufpool.Put(buf)
	return err
}

// creates a new stream buffer and starts a goroutine to monitor
func (c *coalescer) getStreamBuffer(uuid UUID) *streamBuffer {
	return nil
}

type streamBuffer struct {
	readings []Reading
	idx      int
	uuid     UUID
	stop     *time.Timer
	sync.RWMutex
}

func (sb *streamBuffer) fits(readings []Reading) bool {
	return sb.idx+len(readings) < COALESCE_MAX
}

// copy the readings into the buffer to be committed. Returns true if the
// buffer is ready to be deleted, false otherwise
func (sb *streamBuffer) add(readings []Reading) bool {
	// grab write lock and append readings to the buffer
	sb.Lock()
	copied := copy(sb.readings[sb.idx:], readings)
	sb.idx += copied
	sb.Unlock()

	// grab read lock and test that we aren't already full
	sb.RLock()
	if len(sb.readings) >= COALESCE_MAX {
		sb.RUnlock()
		return true
	}
	sb.RUnlock()
	return false
}
