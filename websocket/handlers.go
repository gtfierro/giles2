package websocket

import (
	"github.com/gorilla/websocket"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"net"
	"net/http"
	"os"
	"strconv"
)

// logger
var log *logging.Logger

// set up logging facilities
func init() {
	log = logging.MustGetLogger("websocket")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WebSocketHandler struct {
	a       *giles.Archiver
	handler http.Handler
}

func NewWebSocketHandler(a *giles.Archiver) *WebSocketHandler {
	r := httprouter.New()
	h := &WebSocketHandler{a, r}
	r.GET("/add/:key", h.handleAdd)
	r.GET("/republish", h.handleRepublish)

	go m.start()

	return h
}

func Handle(a *giles.Archiver, port int) {
	h := NewWebSocketHandler(a)

	address, err := net.ResolveTCPAddr("tcp4", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		log.Fatal("Error resolving address %v: %v", "0.0.0.0:"+strconv.Itoa(port), err)
	}

	log.Notice("Starting WebSockets on %v", address.String())
	srv := &http.Server{
		Addr:    address.String(),
		Handler: h.handler,
	}
	srv.ListenAndServe()
}

func (h *WebSocketHandler) handleAdd(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		ephkey   giles.EphemeralKey
		messages giles.TieredSmapMessage
		err      error
	)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Error("Error establishing websocket: %v", err)
		return
	}
	copy(ephkey[:], ps.ByName("key"))

	for {
		err = ws.ReadJSON(messages)
		if err != nil {
			log.Error("Error reading JSON: %v", err)
			ws.Close()
			return
		}
		log.Debug("got %v", messages)
	}

}

func (h *WebSocketHandler) handleRepublish(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		ephkey giles.EphemeralKey
		//messages giles.TieredSmapMessage
		err error
	)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Error("Error establishing websocket: %v", err)
		return
	}
	copy(ephkey[:], ps.ByName("key"))
	msgtype, msg, err := ws.ReadMessage()
	log.Debug("msgtype: %v, msg: %v, err: %v, ephkey: %v", msgtype, string(msg), err, ephkey)

	subscription := StartSubscriber(ws)
	h.a.HandleNewSubscriber(subscription, "select * where "+string(msg), ephkey)
}

type manager struct {
	// registered connections
	subscribers map[*WebSocketSubscriber]bool

	// new connection request
	initialize chan *WebSocketSubscriber

	// get rid of old connections
	remove chan *WebSocketSubscriber
}

var m = manager{
	subscribers: make(map[*WebSocketSubscriber]bool),
	initialize:  make(chan *WebSocketSubscriber),
	remove:      make(chan *WebSocketSubscriber),
}

func (m *manager) start() {
	for {
		select {
		case s := <-m.initialize:
			m.subscribers[s] = true
		case s := <-m.remove:
			if _, found := m.subscribers[s]; found {
				s.closed = true
				s.notify <- true
				delete(m.subscribers, s)
				//close(s.outbound)
			}
		}
	}
}
