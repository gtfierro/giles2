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
	rl := qsr.NewRecordList(qr.seg, len(buf.readings))
	rla := rl.ToArray()
	for i, val := range buf.readings {
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
	if _, err = q.receive(conn, -1); err != nil {
		return fmt.Errorf("Error writing to quasar %v", err)
	}
	q.packetpool.Put(qr)
	return nil
}

func (q *quasarDB) Prev([]UUID, uint64, UnitOfTime) ([]SmapNumbersResponse, error) {
	return nil, nil
}

func (q *quasarDB) Next([]UUID, uint64, UnitOfTime) ([]SmapNumbersResponse, error) {
	return nil, nil
}

func (q *quasarDB) GetData([]UUID, uint64, uint64, UnitOfTime) ([]SmapNumbersResponse, error) {
	return nil, nil
}

func (q *quasarDB) receive(conn *tsConn, limit int32) (SmapNumbersResponse, error) {
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
		log.Debug("limit %v, num values %v", limit, len(resp.Records().Values().ToArray()))
		for i, rec := range resp.Records().Values().ToArray() {
			if limit > -1 && int32(i) >= limit {
				break
			}
			sr.Readings = append(sr.Readings, &SmapNumberReading{Time: uint64(rec.Time()), Value: rec.Value()})
		}
		return sr, nil
	default:
		return sr, fmt.Errorf("Got unexpected Quasar Error code (%v)", resp.StatusCode().String())
	}
	return sr, nil

}
