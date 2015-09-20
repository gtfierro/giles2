package archiver

import (
	"bytes"
	"fmt"
	capn "github.com/glycerine/go-capnproto"
	qsr "github.com/gtfierro/giles2/archiver/quasarcapnp"
	"github.com/satori/go.uuid"
	"net"
	"sync"
)

type quasarDB struct {
	addr           *net.TCPAddr
	mdStore        MetadataStore
	maxConnections int
	packetpool     sync.Pool
	bufferpool     sync.Pool
	connpool       *connectionPool
}

type quasarConfig struct {
	addr           *net.TCPAddr
	mdStore        MetadataStore
	maxConnections int
}

type quasarReading struct {
	seg *capn.Segment
	req *qsr.Request
	ins *qsr.CmdInsertValues
}

func newQuasarDB(c *quasarConfig) *quasarDB {
	q := &quasarDB{
		addr:           c.addr,
		mdStore:        c.mdStore,
		maxConnections: c.maxConnections,
	}
	log.Notice("Connecting to Quasar at %v...", q.addr.String())
	q.packetpool = sync.Pool{
		New: func() interface{} {
			seg := capn.NewBuffer(nil)
			req := qsr.NewRootRequest(seg)
			req.SetEchoTag(0)
			ins := qsr.NewCmdInsertValues(seg)
			ins.SetSync(false)
			return quasarReading{
				seg: seg,
				req: &req,
				ins: &ins,
			}
		},
	}
	q.bufferpool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 200)) // 200 byte buffer
		},
	}
	q.connpool = NewConnectionPool(q.getConnection, q.maxConnections)
	return q
}

func (q *quasarDB) getConnection() *tsConn {
	conn, err := net.DialTCP("tcp", nil, q.addr)
	if err != nil {
		log.Error("Error getting connection to Quasar (%v)", err)
		return nil
	}
	conn.SetKeepAlive(true)
	return &tsConn{conn, false}
}

func (q *quasarDB) AddMessage(msg *SmapMessage) error {
	var (
		parsed_uuid uuid.UUID
		err         error
	)
	if len(msg.Readings) == 0 {
		return nil
	}
	conn := q.connpool.Get()
	defer q.connpool.Put(conn)
	if parsed_uuid, err = uuid.FromString(string(msg.UUID)); err != nil {
		return err
	}
	qr := q.packetpool.Get().(quasarReading)
	qr.ins.SetUuid(parsed_uuid.Bytes())
	rl := qsr.NewRecordList(qr.seg, len(msg.Readings))
	rla := rl.ToArray()
	for i, val := range msg.Readings {
		rla[i].SetTime(int64(val.GetTime()))
		if num, ok := val.GetValue().(float64); ok {
			rla[i].SetValue(num)
		} else {
			return fmt.Errorf("Bad number in message %v %v", msg.UUID, val)
		}
	}
	qr.ins.SetValues(rl)
	qr.req.SetInsertValues(*qr.ins)
	qr.seg.WriteTo(conn)
	if _, err = q.receive(conn); err != nil {
		return fmt.Errorf("Error writing to quasar %v", err)
	}
	q.packetpool.Put(qr)
	return nil
}

func (q *quasarDB) AddBuffer(buf *streamBuffer) error {
	var (
		parsed_uuid uuid.UUID
		err         error
	)
	if len(buf.readings) == 0 {
		return nil
	}
	conn := q.connpool.Get()
	defer q.connpool.Put(conn)
	if parsed_uuid, err = uuid.FromString(string(buf.uuid)); err != nil {
		return err
	}
	qr := q.packetpool.Get().(quasarReading)
	qr.ins.SetUuid(parsed_uuid.Bytes())
	rl := qsr.NewRecordList(qr.seg, buf.idx)
	rla := rl.ToArray()
	for i, val := range buf.readings[:buf.idx] {
		rla[i].SetTime(int64(val.GetTime()))
		if num, ok := val.GetValue().(float64); ok {
			rla[i].SetValue(num)
		} else {
			return fmt.Errorf("Bad number in buffer %v %v", buf.uuid, val)
		}
	}
	qr.ins.SetValues(rl)
	qr.req.SetInsertValues(*qr.ins)
	qr.seg.WriteTo(conn)
	if _, err = q.receive(conn); err != nil {
		return fmt.Errorf("Error writing to quasar %v", err)
	}
	q.packetpool.Put(qr)
	return nil
}

func (quasar *quasarDB) queryNearestValue(uuids []UUID, start uint64, backwards bool) ([]SmapNumbersResponse, error) {
	var ret = make([]SmapNumbersResponse, len(uuids))
	conn := quasar.connpool.Get()
	defer quasar.connpool.Put(conn)
	for i, uu := range uuids {
		seg := capn.NewBuffer(nil)
		req := qsr.NewRootRequest(seg)
		qnv := qsr.NewCmdQueryNearestValue(seg)
		qnv.SetBackward(backwards)
		uuid, _ := uuid.FromString(string(uu))
		qnv.SetUuid(uuid.Bytes())
		qnv.SetTime(int64(start))
		req.SetQueryNearestValue(qnv)
		_, err := seg.WriteTo(conn) // here, ignoring # bytes written
		if err != nil {
			return ret, err
		}
		sr, err := quasar.receive(conn)
		if err != nil {
			return ret, err
		}
		sr.UUID = uu
		ret[i] = sr
	}
	return ret, nil
}

func (q *quasarDB) Prev(uuids []UUID, start uint64) ([]SmapNumbersResponse, error) {
	return q.queryNearestValue(uuids, start, true)
}

func (q *quasarDB) Next(uuids []UUID, start uint64) ([]SmapNumbersResponse, error) {
	return q.queryNearestValue(uuids, start, false)
}

func (q *quasarDB) GetData(uuids []UUID, start uint64, end uint64) ([]SmapNumbersResponse, error) {
	var ret = make([]SmapNumbersResponse, len(uuids))
	conn := q.connpool.Get()
	defer q.connpool.Put(conn)
	for i, uu := range uuids {
		seg := capn.NewBuffer(nil)
		req := qsr.NewRootRequest(seg)
		qnv := qsr.NewCmdQueryStandardValues(seg)
		uuid, _ := uuid.FromString(string(uu))
		qnv.SetUuid(uuid.Bytes())
		qnv.SetStartTime(int64(start))
		qnv.SetEndTime(int64(end))
		req.SetQueryStandardValues(qnv)
		_, err := seg.WriteTo(conn) // here, ignoring # bytes written
		if err != nil {
			return ret, err
		}
		sr, err := q.receive(conn)
		if err != nil {
			return ret, err
		}
		sr.UUID = uu
		ret[i] = sr
	}
	return ret, nil
}

func (q *quasarDB) receive(conn *tsConn) (SmapNumbersResponse, error) {
	var sr = SmapNumbersResponse{}
	seg, err := capn.ReadFromStream(conn, nil)
	if err != nil {
		conn.Close()
		log.Error("Error receiving data from Quasar %v", err)
		return sr, err
	}
	resp := qsr.ReadRootResponse(seg)

	switch resp.Which() {
	case qsr.RESPONSE_VOID:
		if resp.StatusCode() != qsr.STATUSCODE_OK {
			return sr, fmt.Errorf("Received error status code when writing: %v", resp.StatusCode())
		}
	case qsr.RESPONSE_RECORDS:
		if resp.StatusCode() != 0 {
			return sr, fmt.Errorf("Error when reading from Quasar: %v", resp.StatusCode().String())
		}
		sr.Readings = []*SmapNumberReading{}
		for _, rec := range resp.Records().Values().ToArray() {
			sr.Readings = append(sr.Readings, &SmapNumberReading{Time: uint64(rec.Time()), Value: rec.Value()})
		}
		return sr, nil
	default:
		return sr, fmt.Errorf("Got unexpected Quasar Error code (%v)", resp.StatusCode().String())
	}
	return sr, nil

}
