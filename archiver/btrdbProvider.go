package archiver

import (
	"fmt"
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
	b := &btrdbDB{
		addr:           c.addr,
		mdStore:        c.mdStore,
		maxConnections: c.maxConnections,
	}

	log.Notice("Connecting to BtrDB at %v...", b.addr.String())
	// check for liveliness
	if tmp := b.getConnection(); tmp == nil {
		log.Fatal("BtrDB instance not found")
	}

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

	b.connpool = NewConnectionPool(b.getConnection, b.maxConnections)
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
	var sr = SmapNumbersResponse{}
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
		return fmt.Errorf("Received error status code when writing: %v", resp.StatusCode())
	} else {
		return fmt.Errorf("Received a non-VOID response %v. Probably receiveData was intended?", resp.Which())
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
	return nil
}

func (b *btrdbDB) Prev(uuids []UUID, start uint64) ([]SmapNumbersResponse, error) {
	return make([]SmapNumbersResponse, 1), nil
}

func (b *btrdbDB) Next(uuids []UUID, start uint64) ([]SmapNumbersResponse, error) {
	return make([]SmapNumbersResponse, 1), nil
}

func (b *btrdbDB) GetData(uuids []UUID, start, end uint64) ([]SmapNumbersResponse, error) {
	return make([]SmapNumbersResponse, 1), nil
}
