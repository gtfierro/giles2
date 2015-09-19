// Package msgpack implements an MsgPack/UDP interface to the Archiver API at
// http://godoc.org/github.com/gtfierro/2giles/archiver
//
// An example of a valid object is a msgpack map (fixmap or otherwise) with the following
// fields:
//  map:
//      path => message
//  message:
//      metadata => map (flat key/value)
//      properties => map (flat key/value)
//      readings => array
//      uuid => string?
package msgpack

import (
	"github.com/gtfierro/giles2/archiver"
	"github.com/op/go-logging"
	"gopkg.in/vmihailenco/msgpack.v2"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// logger
var log *logging.Logger

// set up logging facilities
func init() {
	log = logging.MustGetLogger("archiver")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type MsgPackUdpHandler struct {
	a       *archiver.Archiver
	bufpool sync.Pool
	msgpool sync.Pool
	counter uint64
}

func HandleUDP4(a *archiver.Archiver, port int) {
	h := &MsgPackUdpHandler{
		a: a,
		bufpool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
		msgpool: sync.Pool{
			New: func() interface{} {
				return &archiver.SmapMessage{
					Actuator: make(archiver.Dict),
					Metadata: make(archiver.Dict),
					Readings: make([]archiver.Reading, 0),
				}
			},
		},
		counter: 0,
	}
	go func() {
		var t = time.NewTicker(1 * time.Second)
		for {
			select {
			case <-t.C:
				log.Info("Pkt %v", atomic.LoadUint64(&h.counter))
				atomic.StoreUint64(&h.counter, 0)
			}
		}
	}()
	udpAddr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		log.Fatal("Error resolving UDP address for msgpack %v", err)
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal("Error on listening (%v)", err)
	}

	log.Notice("Starting MsgPack on UDP %v", udpAddr.String())

	for {
		buf := h.bufpool.Get().([]byte)
		num, from, err := conn.ReadFromUDP(buf)
		go h.handleAdd(buf, num, from, err)
	}
}

func (h *MsgPackUdpHandler) handleAdd(buffer []byte, num int, from *net.UDPAddr, err error) {
	if err != nil {
		log.Debug("Got err handling MsgPack packet", err)
	}
	//log.Notice("num %v, from %v, buf %v", num, from, string(buffer))

	msg, ephkey := h.decode(buffer)
	h.a.AddData(msg, ephkey)
	atomic.AddUint64(&h.counter, 1)
	h.msgpool.Put(msg)
	h.bufpool.Put(buffer)
}

func (h *MsgPackUdpHandler) decode(buffer []byte) (*archiver.SmapMessage, archiver.EphemeralKey) {
	var (
		ephkey archiver.EphemeralKey
		msgMap map[string]interface{}
	)
	msg := h.msgpool.Get().(*archiver.SmapMessage)
	err := msgpack.Unmarshal(buffer, &msgMap)
	if err != nil {
		log.Error("Error decoding msgpack %v", err)
		return nil, ephkey
	}
	log.Debug("got msg %v", msgMap)
	msg.Path = msgMap["Path"].(string)
	msg.UUID = archiver.UUID(msgMap["uuid"].(string))
	//msg.Metadata = archiver.Dict(msgMap["Metadata"].(map[string]string))
	//msg.Readings[0] = *archiver.SmapNumberReading{}
	log.Debug("got msg %#v", msg)
	//  map:
	//      path => message
	//  message:
	//      metadata => map (flat key/value)
	//      properties => map (flat key/value)
	//      readings => array
	//      uuid => string?
	return msg, ephkey
}
