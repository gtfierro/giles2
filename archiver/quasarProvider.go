package archiver

import (
	"bytes"
	capn "github.com/glycerine/go-capnproto"
	qsr "github.com/gtfierro/giles2/archiver/quasarcapnp"
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
