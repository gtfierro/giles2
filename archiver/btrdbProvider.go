package archiver

import (
	"fmt"
	qtree "github.com/SoftwareDefinedBuildings/btrdb/qtree"
	capn "github.com/glycerine/go-capnproto"
	btrdb "github.com/gtfierro/giles2/archiver/btrdbcapnp"
	"github.com/satori/go.uuid"
	"net"
	"sync"
)

type btrdbDB struct {
	addr           *net.TCPAddr
	mdStore        MetadataStore
	maxConnections int
	packetpool     sync.Pool
	connpool       *connectionPool
}

type btrdbConfig struct {
	addr           *net.TCPAddr
	mdStore        MetadataStore
	maxConnections int
}

type btrdbReading struct {
	seg *capn.Segment
	req *btrdb.Request
	ins *btrdb.CmdInsertValues
}

func newBtrDB(c *btrdbConfig) *btrdbDB {
	var err error
	b := &btrdbDB{
		addr:           c.addr,
		mdStore:        c.mdStore,
		maxConnections: c.maxConnections,
	}

	log.Notice("Connecting to BtrDB at %v...", b.addr.String())

	b.packetpool = sync.Pool{
		New: func() interface{} {
			seg := capn.NewBuffer(nil)
			req := btrdb.NewRootRequest(seg)
			req.SetEchoTag(0)
			ins := btrdb.NewCmdInsertValues(seg)
			ins.SetSync(false)
			return btrdbReading{
				seg: seg,
				req: &req,
				ins: &ins,
			}
		},
	}

	if b.connpool, err = NewConnectionPool(b.getConnection, b.maxConnections); err != nil {
		log.Fatal(err)
	}
	return b
}

func (b *btrdbDB) getConnection() *tsConn {
	conn, err := net.DialTCP("tcp", nil, b.addr)
	if err != nil {
		log.Error("Error getting connection to BtrDB (%v)", err)
		return nil
	}
	conn.SetKeepAlive(true)
	return &tsConn{conn, false}
}

func (b *btrdbDB) receiveData(conn *tsConn) (SmapNumbersResponse, error) {
	var (
		sr       = SmapNumbersResponse{}
		finished = false
	)
	sr.Readings = []*SmapNumberReading{}

	for !finished {
		// wait for response on given connection
		seg, err := capn.ReadFromStream(conn, nil)
		if err != nil {
			conn.Close()
			log.Error("Error receiving data from BtrDB %v", err)
			return sr, err
		}
		resp := btrdb.ReadRootResponse(seg)
		switch resp.Which() {
		case btrdb.RESPONSE_VOID:
			return sr, fmt.Errorf("Got a RESPONSE_VOID with statuscode %v, but was expecting data", resp.StatusCode().String())
		case btrdb.RESPONSE_RECORDS:
			if resp.StatusCode() != btrdb.STATUSCODE_OK {
				return sr, fmt.Errorf("Error when reading from BtrDB: %v", resp.StatusCode().String())
			}
			for _, rec := range resp.Records().Values().ToArray() {
				sr.Readings = append(sr.Readings, &SmapNumberReading{Time: uint64(rec.Time()), Value: rec.Value()})
			}
			finished = resp.Final()
		default:
			log.Error("Got unexpected type: %v with status code %v", resp.Which(), resp.StatusCode().String())
		}
	}
	return sr, nil
}

func (b *btrdbDB) receiveStatus(conn *tsConn) error {
	// wait for response on the given connection
	seg, err := capn.ReadFromStream(conn, nil)
	if err != nil {
		conn.Close()
		log.Error("Error receiving data from BtrDB %v", err)
		return err
	}
	resp := btrdb.ReadRootResponse(seg)

	// react to the type of message
	if resp.Which() == btrdb.RESPONSE_VOID && resp.StatusCode() != btrdb.STATUSCODE_OK {
		return fmt.Errorf("Received error status code when writing: %v", resp.StatusCode().String())
	} else {
		return fmt.Errorf("Received a non-VOID response %v with statuscode %v. Probably receiveData was intended?", resp.Which(), resp.StatusCode().String())
	}
}

func (b *btrdbDB) AddMessage(msg *SmapMessage) error {
	var (
		parsed_uuid uuid.UUID
		err         error
	)

	// fetch the connection we're going to use
	conn := b.connpool.Get()
	defer b.connpool.Put(conn)

	// turn the string representation into UUID bytes
	if parsed_uuid, err = uuid.FromString(string(msg.UUID)); err != nil {
		return err
	}

	// fetch a mostly preallocated packet from the pool
	pkt := b.packetpool.Get().(btrdbReading)
	// set the UUID
	pkt.ins.SetUuid(parsed_uuid.Bytes())
	// allocate space for the readings we're going to commit
	records := btrdb.NewRecordList(pkt.seg, len(msg.Readings))
	// insert readings
	recordsArr := records.ToArray()
	for i, val := range msg.Readings {
		recordsArr[i].SetTime(int64(val.GetTime()))
		if num, ok := val.GetValue().(float64); ok {
			recordsArr[i].SetValue(num)
		} else {
			return fmt.Errorf("Bad number in message %v %v", msg.UUID, val)
		}
	}
	pkt.ins.SetValues(records)

	// set packet type
	pkt.req.SetInsertValues(*pkt.ins)

	// write to the database
	pkt.seg.WriteTo(conn)
	b.packetpool.Put(pkt)
	if err = b.receiveStatus(conn); err != nil {
		return fmt.Errorf("Error writing to btrdb %v", err)
	}
	return nil
}

func (b *btrdbDB) AddBuffer(buf *streamBuffer) error {
	var (
		parsed_uuid uuid.UUID
		err         error
	)
	if len(buf.readings) == 0 {
		return nil
	}
	conn := b.connpool.Get()
	defer b.connpool.Put(conn)
	if parsed_uuid, err = uuid.FromString(string(buf.uuid)); err != nil {
		return err
	}
	pkt := b.packetpool.Get().(btrdbReading)
	pkt.ins.SetUuid(parsed_uuid.Bytes())
	rl := btrdb.NewRecordList(pkt.seg, buf.idx)
	rla := rl.ToArray()
	for i, val := range buf.readings[:buf.idx] {
		rla[i].SetTime(int64(val.GetTime()))
		if num, ok := val.GetValue().(float64); ok {
			rla[i].SetValue(num)
		} else {
			return fmt.Errorf("Bad number in buffer %v %v", buf.uuid, val)
		}
	}
	pkt.ins.SetValues(rl)
	pkt.req.SetInsertValues(*pkt.ins)
	pkt.seg.WriteTo(conn)
	if err = b.receiveStatus(conn); err != nil {
		return fmt.Errorf("Error writing to btrdb %v", err)
	}
	b.packetpool.Put(pkt)
	return nil
}

func (b *btrdbDB) queryNearestValue(uuids []UUID, start uint64, backwards bool) ([]SmapNumbersResponse, error) {
	var ret = make([]SmapNumbersResponse, len(uuids))
	conn := b.connpool.Get()
	defer b.connpool.Put(conn)
	for i, uu := range uuids {
		seg := capn.NewBuffer(nil)
		req := btrdb.NewRootRequest(seg)
		query := btrdb.NewCmdQueryNearestValue(seg)
		query.SetBackward(backwards)
		uuid, _ := uuid.FromString(string(uu))
		query.SetUuid(uuid.Bytes())
		query.SetTime(int64(start))
		req.SetQueryNearestValue(query)
		_, err := seg.WriteTo(conn) // here, ignoring # bytes written
		if err != nil {
			return ret, err
		}
		sr, err := b.receiveData(conn)
		if err != nil {
			return ret, err
		}
		sr.UUID = uu
		ret[i] = sr
	}
	return ret, nil
}

func (b *btrdbDB) Prev(uuids []UUID, start uint64) ([]SmapNumbersResponse, error) {
	return b.queryNearestValue(uuids, start, true)
}

func (b *btrdbDB) Next(uuids []UUID, start uint64) ([]SmapNumbersResponse, error) {
	return b.queryNearestValue(uuids, start, false)
}

func (b *btrdbDB) GetData(uuids []UUID, start, end uint64) ([]SmapNumbersResponse, error) {
	var ret = make([]SmapNumbersResponse, len(uuids))
	conn := b.connpool.Get()
	defer b.connpool.Put(conn)
	for i, uu := range uuids {
		seg := capn.NewBuffer(nil)
		req := btrdb.NewRootRequest(seg)
		query := btrdb.NewCmdQueryStandardValues(seg)
		uuid, _ := uuid.FromString(string(uu))
		query.SetUuid(uuid.Bytes())
		query.SetStartTime(int64(start))
		query.SetEndTime(int64(end))
		req.SetQueryStandardValues(query)
		_, err := seg.WriteTo(conn) // here, ignoring # bytes written
		if err != nil {
			return ret, err
		}
		sr, err := b.receiveData(conn)
		if err != nil {
			return ret, err
		}
		sr.UUID = uu
		ret[i] = sr
	}
	return ret, nil
}

func (b *btrdbDB) ValidTimestamp(time uint64, uot UnitOfTime) bool {
	var err error
	if uot != UOT_NS {
		time, err = convertTime(time, uot, UOT_NS)
	}
	return time >= 0 && time <= qtree.MaximumTime && err == nil
}
