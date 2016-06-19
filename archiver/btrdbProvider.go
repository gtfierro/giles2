package archiver

import (
	"errors"
	"fmt"
	capn "github.com/glycerine/go-capnproto"
	btrdb "github.com/gtfierro/giles2/archiver/btrdbcapnp"
	"github.com/gtfierro/giles2/common"
	"github.com/satori/go.uuid"
	"net"
	"sync"
)

var BtrDBReadErr = errors.New("Error receiving data from BtrDB")

const MaximumTime = (48 << 56)

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

	log.Noticef("Connecting to BtrDB at %v...", b.addr.String())

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
		log.Errorf("Error getting connection to BtrDB (%v)", err)
		return nil
	}
	conn.SetKeepAlive(true)
	return &tsConn{conn, false}
}

func (b *btrdbDB) receiveData(conn *tsConn) (common.SmapNumbersResponse, error) {
	var (
		sr       = common.SmapNumbersResponse{}
		finished = false
	)
	sr.Readings = []*common.SmapNumberReading{}

	for !finished {
		// wait for response on given connection
		seg, err := capn.ReadFromStream(conn, nil)
		if err != nil {
			conn.Close()
			log.Errorf("Error receiving data from BtrDB %v", err)
			return sr, BtrDBReadErr
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
				sr.Readings = append(sr.Readings, &common.SmapNumberReading{Time: uint64(rec.Time()), Value: rec.Value(), UoT: common.UOT_NS})
			}
			finished = resp.Final()
		default:
			log.Errorf("Got unexpected type: %v with status code %v", resp.Which(), resp.StatusCode().String())
		}
	}
	return sr, nil
}

func (b *btrdbDB) receiveStatus(conn *tsConn) error {
	// wait for response on the given connection
	seg, err := capn.ReadFromStream(conn, nil)
	if err != nil {
		conn.Close()
		log.Errorf("Error receiving data from BtrDB %v", err)
		return BtrDBReadErr
	}
	resp := btrdb.ReadRootResponse(seg)

	// react to the type of message
	if resp.Which() == btrdb.RESPONSE_VOID && resp.StatusCode() != btrdb.STATUSCODE_OK {
		return fmt.Errorf("Received error status code when writing: %v", resp.StatusCode().String())
	} else {
		return fmt.Errorf("Received a non-VOID response %v with statuscode %v. Probably receiveData was intended?", resp.Which(), resp.StatusCode().String())
	}
}

func (b *btrdbDB) AddMessage(msg *common.SmapMessage) error {
	var (
		parsed_uuid uuid.UUID
		err         error
	)

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
		val.ConvertTime(common.UOT_NS)
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
	err = b.reliableWriteStatus(&pkt)
	b.packetpool.Put(pkt)
	return err
}

func (b *btrdbDB) queryNearestValue(uuids []common.UUID, start uint64, backwards bool) ([]common.SmapNumbersResponse, error) {
	var ret = make([]common.SmapNumbersResponse, len(uuids))
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

func (b *btrdbDB) Prev(uuids []common.UUID, start uint64) ([]common.SmapNumbersResponse, error) {
	return b.queryNearestValue(uuids, start, true)
}

func (b *btrdbDB) Next(uuids []common.UUID, start uint64) ([]common.SmapNumbersResponse, error) {
	return b.queryNearestValue(uuids, start, false)
}

func (b *btrdbDB) GetData(uuids []common.UUID, start, end uint64) ([]common.SmapNumbersResponse, error) {
	var ret = make([]common.SmapNumbersResponse, len(uuids))
	for i, uu := range uuids {
		seg := capn.NewBuffer(nil)
		req := btrdb.NewRootRequest(seg)
		query := btrdb.NewCmdQueryStandardValues(seg)
		uuid, _ := uuid.FromString(string(uu))
		query.SetUuid(uuid.Bytes())
		query.SetStartTime(int64(start))
		query.SetEndTime(int64(end))
		req.SetQueryStandardValues(query)
		sr, err := b.reliableWriteData(seg)
		if err != nil {
			return ret, err
		}
		sr.UUID = uu
		ret[i] = sr
	}
	return ret, nil
}

func (b *btrdbDB) DeleteData(uuids []common.UUID, start, end uint64) error {
	for _, uu := range uuids {
		seg := capn.NewBuffer(nil)
		req := btrdb.NewRootRequest(seg)
		del := btrdb.NewCmdDeleteValues(seg)
		uuid, _ := uuid.FromString(string(uu))
		del.SetUuid(uuid.Bytes())
		del.SetStartTime(int64(start))
		del.SetEndTime(int64(end))
		req.SetDeleteValues(del)
		_, err := b.reliableWriteData(seg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *btrdbDB) ValidTimestamp(time uint64, uot common.UnitOfTime) bool {
	var err error
	if uot != common.UOT_NS {
		time, err = common.ConvertTime(time, uot, common.UOT_NS)
	}
	return time >= 0 && time <= MaximumTime && err == nil
}

func (b *btrdbDB) reliableWriteStatus(pkt *btrdbReading) error {
	var (
		conn *tsConn
		err  error
	)
	for {
		conn = b.connpool.Get()
		if !conn.IsClosed() {
			pkt.seg.WriteTo(conn)
			if err = b.receiveStatus(conn); err == BtrDBReadErr {
				conn.Close()
				b.connpool.Put(conn)
				//fmt.Errorf("Error writing to btrdb %v", err)
				continue
			} else if err != nil { // if not read error
				b.connpool.Put(conn)
				return err
			}
		}
		break
	}
	b.connpool.Put(conn)
	return nil
}

func (b *btrdbDB) reliableWriteData(seg *capn.Segment) (common.SmapNumbersResponse, error) {
	var (
		sr   common.SmapNumbersResponse
		conn *tsConn
		err  error
	)
	for {
		conn = b.connpool.Get()
		if !conn.IsClosed() {
			seg.WriteTo(conn)
			if sr, err = b.receiveData(conn); err == BtrDBReadErr {
				conn.Close()
				b.connpool.Put(conn)
				//fmt.Errorf("Error writing to btrdb %v", err)
				continue
			} else if err != nil { // if not read error
				b.connpool.Put(conn)
				return sr, err
			}
		}
		break
	}
	b.connpool.Put(conn)
	return sr, nil
}
