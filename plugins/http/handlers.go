// Package http implements an HTTP interface to the Archiver API at
// http://godoc.org/github.com/gtfierro/2giles/archiver
//
// An example of a valid sMAP object is
//    {
//      "/sensor0" : {
//        "Metadata" : {
//          "SourceName" : "Test Source",
//            "Location" : { "City" : "Berkeley" }
//        },
//          "Properties": {
//            "Timezone": "America/Los_Angeles",
//            "UnitofMeasure": "Watt",
//            "UnitofTime": "ms",
//            "StreamType": "numeric",
//            "ReadingType": "double"
//          },
//          "Readings" : [[1351043674000, 0], [1351043675000, 1]],
//          "uuid" : "d24325e6-1d7d-11e2-ad69-a7c2fa8dba61"
//      }
//    }
package http

import (
	"encoding/json"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"io"
	"net"
	"net/http"
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

type HTTPHandler struct {
	a       *giles.Archiver
	handler http.Handler
}

func NewHTTPHandler(a *giles.Archiver) *HTTPHandler {
	r := httprouter.New()
	h := &HTTPHandler{a, r}
	r.POST("/add/:key", h.handleAdd)
	r.POST("/api/query/:key", h.handleSingleQuery)
	r.POST("/api/query", h.handleSingleQuery)
	r.POST("/republish", h.handleRepublisher)
	r.POST("/republish/:key", h.handleRepublisher)
	r.POST("/subscribe", h.handleSubscriber)
	r.POST("/subscribe/:key", h.handleSubscriber)
	return h
}

func Handle(a *giles.Archiver, port int) {
	h := NewHTTPHandler(a)
	address, err := net.ResolveTCPAddr("tcp4", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Error resolving address %v: %v", "0.0.0.0:"+strconv.Itoa(port), err)
	}
	http.Handle("/", h.handler)
	log.Noticef("Starting HTTP on %v", address.String())

	srv := &http.Server{
		Addr: address.String(),
	}
	srv.ListenAndServe()
}

func (h *HTTPHandler) handleAdd(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		messages common.TieredSmapMessage
		err      error
	)
	defer req.Body.Close()

	if messages, err = handleJSON(req.Body); err != nil {
		log.Errorf("Error handling JSON: %v", err)
		rw.WriteHeader(400)
		rw.Write([]byte(err.Error()))
		return
	}

	messages.CollapseToTimeseries()
	for _, msg := range messages {
		if addErr := h.a.AddData(msg); addErr != nil {
			rw.WriteHeader(500)
			rw.Write([]byte(addErr.Error()))
			return
		}
	}

	rw.WriteHeader(200)
}

func (h *HTTPHandler) handleSingleQuery(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		err error
	)
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")

	defer req.Body.Close()

	if req.ContentLength > 1024 {
		log.Errorf("HUGE query string with length %v. Aborting!", req.ContentLength)
		rw.WriteHeader(500)
		rw.Write([]byte("Your query is too big"))
		return
	}

	querybuffer := make([]byte, req.ContentLength)
	_, err = req.Body.Read(querybuffer)
	res, err := h.a.HandleQuery(string(querybuffer))
	if err != nil {
		log.Errorf("Error evaluating query: %v", err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}
	writer := json.NewEncoder(rw)
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = writer.Encode(res)
	if err != nil {
		log.Errorf("Error converting query results to JSON: %v", err)
	}
}

func (h *HTTPHandler) handleSubscriber(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		err error
	)
	defer req.Body.Close()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")

	if req.ContentLength > 1024 {
		log.Errorf("HUGE query string with length %v. Aborting!", req.ContentLength)
		rw.WriteHeader(500)
		rw.Write([]byte("Your query is too big"))
		return
	}

	querybuffer := make([]byte, req.ContentLength)
	_, err = req.Body.Read(querybuffer)
	if err != nil && err.Error() != "EOF" {
		log.Errorf("Error reading subscription: %v", err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}

	subscription := StartHTTPSubscriber(rw)

	h.a.HandleNewSubscriber(subscription, string(querybuffer))
}

func (h *HTTPHandler) handleRepublisher(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		err error
	)
	defer req.Body.Close()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")

	if req.ContentLength > 1024 {
		log.Errorf("HUGE query string with length %v. Aborting!", req.ContentLength)
		rw.WriteHeader(500)
		rw.Write([]byte("Your query is too big"))
		return
	}

	querybuffer := make([]byte, req.ContentLength)
	_, err = req.Body.Read(querybuffer)
	if err != nil && err.Error() != "EOF" {
		log.Errorf("Error reading subscription: %v", err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}

	subscription := StartHTTPSubscriber(rw)

	h.a.HandleNewSubscriber(subscription, "select * where "+string(querybuffer))
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
