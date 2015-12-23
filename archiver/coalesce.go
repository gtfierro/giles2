package archiver

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

//TODO: how do we do throttling? need to ensure that buffers are fairly allocated to streams
//TODO: place incoming requests into a queue? where does queue go. Need back pressure

// the transaction coalescer, upon startup, will initialize all connections to the backend timeseries database and keep them in a pool.
// a big problem with the current transaction coalescer is that it does not define clear
// semantics for what happens when a particular stream saturates -- how are connections fairly allocated to the rest of the streams?
// The solution is thus: we will have some large map with a static size that is used to shard the UUID namespace. Then, we can use
// our map approach to storing stream buffers for each individual stream, but hopefully the sharding of the map will decrease
// any lock contention.

const (
	COALESCE_TIMEOUT = 1000  // milliseconds
	COALESCE_MAX     = 16384 // num readings
)

type streamMap map[UUID](*streamBuffer)

type streamBuffer struct {
	incoming   chan *SmapMessage
	uuid       UUID
	unitOfTime UnitOfTime
	readings   []*SmapNumberReading
	txc        *transactionCoalescer
	closed     atomic.Value
	timeout    <-chan time.Time
	abort      chan bool
	num        int64
	idx        int
	sync.Mutex
}

func newStreamBuf(uuid UUID, uot UnitOfTime, txc *transactionCoalescer) *streamBuffer {
	sb := &streamBuffer{uuid: uuid, unitOfTime: uot,
		incoming: txc.chanpool.Get().(chan *SmapMessage),
		num:      0,
		idx:      0,
		txc:      txc,
		readings: txc.bufpool.Get().([]*SmapNumberReading),
		abort:    make(chan bool, 1),
		timeout:  time.After(time.Duration(COALESCE_TIMEOUT) * time.Millisecond)}
	sb.closed.Store(false)
	go sb.watch()
	return sb
}

func (sb *streamBuffer) watch() {
	select {
	case <-sb.timeout:
		sb.commit()
	case <-sb.abort:
	}
}

func (sb *streamBuffer) isClosed() bool {
	return sb.closed.Load().(bool)
}

// Returns true if successfully added SmapMessage to the buffer,
// and false if the buffer is already closed
func (sb *streamBuffer) add(sm *SmapMessage) bool {
	// if no longer accepting readings, return false
	if sb.isClosed() {
		return false
	}

	sb.Lock()
	// if we are short some readings, append the space to the end
	if diff := (len(sm.Readings) + sb.idx) - COALESCE_MAX; diff > 0 {
		sb.readings = append(sb.readings, make([]*SmapNumberReading, diff)...)
	}
	// copy over all the readings
	idx := 0
	for _, rdg := range sm.Readings {
		if rdg == nil {
			continue
		}
		sb.readings[sb.idx+idx] = rdg.(*SmapNumberReading)
		sb.readings[sb.idx+idx].ConvertTime(sb.unitOfTime, UOT_NS)
		idx += 1
	}
	// advance our pointer
	sb.idx += len(sm.Readings)

	if sb.idx >= COALESCE_MAX {
		sb.abort <- true // cancels the timeout
		sb.Unlock()
		sb.commit()
		return true
	}
	sb.Unlock()
	return true
}

func (sb *streamBuffer) commit() {
	// close from further readings
	sb.closed.Store(true)
	// dispatch the commit
	sb.txc.Commit(sb)
}

type transactionCoalescer struct {
	tsdb     *TimeseriesStore
	store    *MetadataStore
	streams  atomic.Value
	bufpool  sync.Pool
	chanpool sync.Pool
	sync.Mutex
}

func newTransactionCoalescer(tsdb *TimeseriesStore, store *MetadataStore) *transactionCoalescer {
	txc := &transactionCoalescer{tsdb: tsdb, store: store}
	txc.streams.Store(make(streamMap))
	txc.bufpool = sync.Pool{
		New: func() interface{} {
			return make([]*SmapNumberReading, COALESCE_MAX)
		},
	}
	txc.chanpool = sync.Pool{
		New: func() interface{} {
			return make(chan *SmapMessage, COALESCE_MAX)
		},
	}
	return txc
}

// Called to add an incoming SmapMessage to the underlying timeseries database. A SmapMessage contains
// an array of Readings and the UUID for the stream the readings belong to. The Readings must be added to
// a StreamBuffer for coalescing. This StreamBuffer is either a) pre-existing and still open, b) pre-existing and committing or
// c) not existing. In the
func (txc *transactionCoalescer) AddSmapMessage(sm *SmapMessage) error {
	var sb *streamBuffer

	if sm.Readings == nil || len(sm.Readings) == 0 {
		return nil
	}

	// if we find the stream buffer and it is still accepting data, we write to that
	// stream and then return
	streams := txc.streams.Load().(streamMap)
	if sb, found := streams[sm.UUID]; found && sb != nil {
		if sb.add(sm) {
			return nil
		}
	}

	txc.Lock()
	streams = txc.streams.Load().(streamMap)
	// check again
	if sb, found := streams[sm.UUID]; found && sb != nil {
		if sb.add(sm) {
			txc.Unlock()
			return nil
		}
	}
	uot, err := (*txc.store).GetUnitOfTime(sm.UUID)
	if err != nil {
		txc.Unlock()
		return err
	}
	if !(*txc.tsdb).ValidTimestamp(sm.Readings[0].GetTime(), uot) {
		txc.Unlock()
		return fmt.Errorf("Bad Timestamp: %v", sm.Readings[0].GetTime())
	}
	sb = newStreamBuf(sm.UUID, uot, txc)
	newStreams := make(streamMap, len(streams)+1)
	for k, v := range streams {
		newStreams[k] = v
	}
	newStreams[sm.UUID] = sb
	txc.streams.Store(newStreams)
	txc.Unlock()
	txc.AddSmapMessage(sm)
	return nil
}

func (txc *transactionCoalescer) Commit(sb *streamBuffer) {
	streams := txc.streams.Load().(streamMap)
	if streams[sb.uuid] == sb {
		txc.Lock()
		newStreams := make(streamMap, len(streams))
		for k, v := range streams {
			newStreams[k] = v
		}
		delete(newStreams, sb.uuid)
		txc.streams.Store(streams)
		txc.Unlock()
	}
	sb.Lock()
	(*txc.tsdb).AddBuffer(sb)
	txc.bufpool.Put(sb.readings)
	txc.chanpool.Put(sb.incoming)
	sb.Unlock()
}
