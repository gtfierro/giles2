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
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	"github.com/op/go-logging"
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
	log = logging.MustGetLogger("msgpack")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type MsgPackUdpHandler struct {
	a       *giles.Archiver
	bufpool sync.Pool
	msgpool sync.Pool
	counter uint64
}

func HandleUDP4(a *giles.Archiver, port int) {
	h := &MsgPackUdpHandler{
		a: a,
		bufpool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
		msgpool: sync.Pool{
			New: func() interface{} {
				return &common.SmapMessage{
					Actuator: make(common.Dict),
					Metadata: make(common.Dict),
					Readings: make([]common.Reading, 0),
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
				atomic.StoreUint64(&h.counter, 0)
			}
		}
	}()
	udpAddr, err := net.ResolveUDPAddr("udp6", "[::]:"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Error resolving UDP address for msgpack %v", err)
	}
	conn, err := net.ListenUDP("udp6", udpAddr)
	if err != nil {
		log.Fatalf("Error on listening (%v)", err)
	}

	log.Noticef("Starting MsgPack on UDP %v", udpAddr.String())

	for {
		buf := h.bufpool.Get().([]byte)
		num, from, err := conn.ReadFromUDP(buf)
		go h.handleAdd(buf, num, from, err)
	}
}

func (h *MsgPackUdpHandler) handleAdd(buffer []byte, num int, from *net.UDPAddr, err error) {
	if err != nil {
		log.Debugf("Got err handling MsgPack packet", err)
		return
	}

	msg, err := h.decode(buffer)
	if err == nil {
		h.a.AddData(msg)
		atomic.AddUint64(&h.counter, 1)
	}
	msg.Metadata = common.Dict{}
	h.msgpool.Put(msg)
	h.bufpool.Put(buffer)
}

func (h *MsgPackUdpHandler) handleSubscription(buffer []byte, num int, from *net.UDPAddr, err error) {
	if err != nil {
		log.Debugf("Got err handling MsgPack packet", err)
		return
	}
}

func Fuzz(data []byte) int {
	h := &MsgPackUdpHandler{
		bufpool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
		msgpool: sync.Pool{
			New: func() interface{} {
				return &common.SmapMessage{
					Actuator: make(common.Dict),
					Metadata: make(common.Dict),
					Readings: make([]common.Reading, 0),
				}
			},
		},
		counter: 0,
	}
	msg, err := h.decode(data)
	if msg == nil || err != nil {
		return 1
	}
	return 0
}

func (h *MsgPackUdpHandler) decode(buffer []byte) (*common.SmapMessage, error) {
	var (
		uuid string
	)
	msg := h.msgpool.Get().(*common.SmapMessage)

	msgMap, err := doDecode(buffer)

	if err != nil {
		log.Errorf("Error decoding msgpack %v", err)
		return nil, err
	}

	// get Path
	if msg.Path, err = getStringValue(msgMap, "Path"); err != nil {
		return msg, err
	}

	// get UUID
	if uuid, err = getStringValue(msgMap, "uuid"); err != nil {
		return msg, err
	}
	msg.UUID = common.UUID(uuid)

	// test for readings
	rdgs, err := getReadings(msgMap)
	if err != nil && err != ReadingsNotFound {
		return msg, err // return early if we found readings and it still gave error
	} else if err == ReadingsNotFound { // otherwise look for Value field
		var value float64
		if value, err = getValue(msgMap); err != nil {
			return msg, err
		}
		msg.Readings = []common.Reading{&common.SmapNumberReading{Time: common.GetNow(common.UOT_MS), Value: value}}
	} else if err == nil { // readings are ok
		msg.Readings = rdgs
	}

	//get Metadata
	md, err := getMetadata(msgMap)
	if err != nil && err != MetadataNotFound {
		return msg, err
	} else if err == nil {
		msg.Metadata = md
	}

	//get Properties
	props, err := getProperties(msgMap)
	if err != nil && err != PropertiesNotFound {
		return msg, err
	} else if err == nil {
		msg.Properties = props
	}

	return msg, nil
}
