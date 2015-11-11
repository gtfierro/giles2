package archiver

//
//import (
//	"bytes"
//	"fmt"
//	capn "github.com/glycerine/go-capnproto"
//	btr "github.com/gtfierro/giles2/archiver/quasarcapnp"
//	"github.com/satori/go.uuid"
//	"net"
//	"sync"
//)
//
//type btrdb struct {
//	addr           *net.TCPAddr
//	mdStore        MetadataStore
//	maxConnections int
//	packetpool     sync.Pool
//	bufferpool     sync.Pool
//	connpool       *connectionPool
//}
//
//type btrdbConfig struct {
//	addr           *net.TCPAddr
//	mdStore        MetadataStore
//	maxConnections int
//}
//
//func newBtrdb(c *btrdbConfig) *btrdb {
//	b := &btrdb{
//		addr:           c.addr,
//		mdStore:        c.mdStore,
//		maxConnections: c.maxConnections,
//	}
//	log.Notice("Connecting to BtrDB at %v...", b.addr.String())
//	b.packetpool = sync.Pool{
//		New: func() interface{} {
//			seg := capn.NewBuffer(nil)
//			req := btr.NewRootRequest(seg)
//			req.SetEchoTag(0)
//			ins := btr.NewCmdInsertValues(seg)
//			ins.SetSync(false)
//			return quasarReading{
//				seg: seg,
//				req: &req,
//				ins: &ins,
//			}
//		},
//	}
//	q.bufferpool = sync.Pool{
//		New: func() interface{} {
//			return bytes.NewBuffer(make([]byte, 0, 200)) // 200 byte buffer
//		},
//	}
//	q.connpool = NewConnectionPool(q.getConnection, q.maxConnections)
//	return q
//}
