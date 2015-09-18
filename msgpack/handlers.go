package msgpack

import (
	"github.com/gtfierro/giles2/archiver"
	"github.com/op/go-logging"
	"net"
	"os"
	"strconv"
	"sync"
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
}

func HandleUDP4(a *archiver.Archiver, port int) {
	h := &MsgPackUdpHandler{
		a: a,
		bufpool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
	}
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
	log.Notice("num %v, from %v, buf %v", num, from, string(buffer))

	h.bufpool.Put(buffer)
}
