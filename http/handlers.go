package http

import (
	"encoding/json"
	"github.com/gtfierro/giles2/archiver"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"io"
	"net"
	"net/http"
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

type HTTPHandler struct {
	a *archiver.Archiver
}

func Handle(a *archiver.Archiver, port int) {
	r := httprouter.New()
	h := &HTTPHandler{a}
	r.POST("/add/:key", h.handleAdd)
	address, err := net.ResolveTCPAddr("tcp4", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		log.Fatal("Error resolving address %v: %v", "0.0.0.0:"+strconv.Itoa(port), err)
	}
	http.Handle("/", r)
	log.Notice("Starting HTTP on %v", address.String())

	srv := &http.Server{
		Addr: address.String(),
	}
	srv.ListenAndServe()
}

func (h *HTTPHandler) handleAdd(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		ephkey   archiver.EphemeralKey
		messages archiver.TieredSmapMessage
		err      error
		msgSync  sync.WaitGroup
	)
	copy(ephkey[:], ps.ByName("key"))

	if messages, err = handleJSON(req.Body); err != nil {
		log.Error("Error handling JSON: %v", err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		req.Body.Close()
		return
	}

	msgSync.Add(len(messages))
	for _, msg := range messages {
		go func(msg *archiver.SmapMessage) {
			if addErr := h.a.AddData(msg, ephkey); addErr != nil {
				err = addErr
			}
			msgSync.Done()
		}(msg)
	}

	msgSync.Wait()
	rw.WriteHeader(200)
}

func handleJSON(r io.Reader) (decoded archiver.TieredSmapMessage, err error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	err = decoder.Decode(&decoded)
	for path, msg := range decoded {
		msg.Path = path
	}
	return
}
