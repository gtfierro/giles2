// Package tcpjson implements a JSON/TCP interfaces for the archiver API.
package tcpjson

import (
	"encoding/json"
	"fmt"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	"github.com/op/go-logging"
	"io"
	"net"
	"os"
	"strconv"
)

// logger
var log *logging.Logger

// set up logging facilities
func init() {
	log = logging.MustGetLogger("http")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type TCPJSONHandler struct {
	a       *giles.Archiver
	errors  chan error
	addAddr *net.TCPAddr
	addConn *net.TCPListener

	queryAddr *net.TCPAddr
	queryConn *net.TCPListener

	subscribeAddr *net.TCPAddr
	subscribeConn *net.TCPListener
}

func NewTCPJSONHandler(a *giles.Archiver, addPort, queryPort, subscribePort int) *TCPJSONHandler {
	var err error
	t := &TCPJSONHandler{a: a, errors: make(chan error)}

	t.addAddr, err = net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(addPort))
	if err != nil {
		log.Fatalf("Error resolving TCPJSON address %v (%v)", addPort, err)
	}
	t.addConn, err = net.ListenTCP("tcp", t.addAddr)
	if err != nil {
		log.Fatalf("Error listening to TCP (%v)", err)
	}

	t.queryAddr, err = net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(queryPort))
	if err != nil {
		log.Fatalf("Error resolving TCPJSON address %v (%v)", queryPort, err)
	}
	t.queryConn, err = net.ListenTCP("tcp", t.queryAddr)
	if err != nil {
		log.Fatalf("Error listening to TCP (%v)", err)
	}

	t.subscribeAddr, err = net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(subscribePort))
	if err != nil {
		log.Fatalf("Error resolving TCPJSON address %v (%v)", subscribePort, err)
	}
	t.subscribeConn, err = net.ListenTCP("tcp", t.subscribeAddr)
	if err != nil {
		log.Fatalf("Error listening to TCP (%v)", err)
	}

	return t
}

func Handle(a *giles.Archiver, addPort, queryPort, subscribePort int) {
	tcp := NewTCPJSONHandler(a, addPort, queryPort, subscribePort)
	go tcp.listenAdds()
	go tcp.listenQuery()
	go tcp.listenSubscribe()
	log.Noticef("Starting JSON/TCP on Add:%v Query:%v Subscribe:%v", addPort, queryPort, subscribePort)
	for err := range tcp.errors {
		log.Error(err)
	}
}

func (tcp *TCPJSONHandler) listenAdds() {
	for {
		conn, err := tcp.addConn.Accept()
		if err != nil {
			tcp.errors <- err
			continue
		}
		go tcp.handleAdd(conn)
	}
}

func (tcp *TCPJSONHandler) handleAdd(conn net.Conn) {
	var (
		messages common.TieredSmapMessage
		err      error
	)
	if messages, err = handleJSON(conn); err != nil {
		log.Errorf("Error handling JSON: %v", err)
		tcp.errors <- err
		return
	}
	messages.CollapseToTimeseries()
	for _, msg := range messages {
		if addErr := tcp.a.AddData(msg); addErr != nil {
			log.Errorf("Error handling JSON: %v", err)
			tcp.errors <- err
			conn.Close()
			return
		}
	}

}

func (tcp *TCPJSONHandler) listenQuery() {
	for {
		conn, err := tcp.queryConn.Accept()
		if err != nil {
			tcp.errors <- err
			continue
		}
		go tcp.handleQuery(conn)
	}
}

func (tcp *TCPJSONHandler) handleQuery(conn net.Conn) {
	defer conn.Close()
	querybuffer := make([]byte, 1024) // shouldn't have a bigger query
	n, err := conn.Read(querybuffer)
	if n == 1024 {
		tcp.errors <- fmt.Errorf("N = 1024 not big enough!")
	} else if err != nil {
		tcp.errors <- err
		return
	}
	res, err := tcp.a.HandleQuery(string(querybuffer))
	if err != nil {
		log.Errorf("Error evaluating query: %v", err)
		tcp.errors <- err
		return
	}
	writer := json.NewEncoder(conn)
	err = writer.Encode(res)
	if err != nil {
		log.Errorf("Error converting query results to JSON: %v", err)
	}
}

func (tcp *TCPJSONHandler) listenSubscribe() {
	for {
		conn, err := tcp.subscribeConn.Accept()
		if err != nil {
			tcp.errors <- err
			continue
		}
		go tcp.handleSubscribe(conn)
	}
}

func (tcp *TCPJSONHandler) handleSubscribe(conn net.Conn) {
	querybuffer := make([]byte, 1024) // shouldn't have a bigger query
	n, err := conn.Read(querybuffer)
	if n == 1024 {
		tcp.errors <- fmt.Errorf("N = 1024 not big enough!")
	} else if err != nil {
		tcp.errors <- err
		return
	}

	subscription := StartTCPJSONSubscriber(conn)
	tcp.a.HandleNewSubscriber(subscription, "select * where "+string(querybuffer))
}

func handleJSON(r io.Reader) (decoded common.TieredSmapMessage, err error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	err = decoder.Decode(&decoded)
	for path, msg := range decoded {
		msg.Path = path
	}
	return
}
