// Client library for communicating with giles using JSON/TCP
package client

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/op/go-logging"
	"net"
	"os"
)

// logger
var log *logging.Logger

// set up logging facilities
func init() {
	log = logging.MustGetLogger("client")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type Client struct {
	QueryAddr     *net.TCPAddr
	SubscribeAddr *net.TCPAddr
	Subscriptions map[string]chan giles.SmapMessage
}

// Creates a new client with connections to the Giles instance at the provided addresses. Its
// on the list to create a better client configuration, but for now, the two arguments here
// use the AddPort and QueryPort config options from `giles.cfg` under the `TCPJSON` section header.
// Remember to use the full addresses, e.g.
//  "1.2.3.4:8002", "1.2.3.4:8003"
func NewClient(queryAddr, subAddr string) *Client {
	qA, err := net.ResolveTCPAddr("tcp", queryAddr)
	if err != nil {
		log.Fatal(err)
	}
	sA, err := net.ResolveTCPAddr("tcp", subAddr)
	if err != nil {
		log.Fatal(err)
	}
	c := &Client{
		QueryAddr:     qA,
		SubscribeAddr: sA,
		Subscriptions: make(map[string]chan giles.SmapMessage),
	}
	return c
}

func (c *Client) doErrorString(messages ...string) {
	bold_yellow := color.New(color.FgYellow, color.Bold)
	bold_yellow.Println("Got error! ===>")
	for _, msg := range messages {
		color.Red("  " + msg)
	}
}

func (c *Client) doError(messages ...error) {
	bold_yellow := color.New(color.FgYellow, color.Bold)
	bold_yellow.Println("Got error! ===>")
	for _, msg := range messages {
		color.Red("  " + msg.Error())
	}
}

func (c *Client) DecodeMessage(conn net.Conn) (decoded *giles.SmapMessageList, err error) {
	decoder := json.NewDecoder(conn)
	decoder.UseNumber()
	err = decoder.Decode(&decoded)
	return
}

// Executes the provided query and returns a list of giles.SmapMessage. This does not yet
// handle different return types such as those returned by 'select distinct'
func (c *Client) Query(query string) *giles.SmapMessageList {
	var decoded *giles.SmapMessageList
	conn, err := net.DialTCP("tcp", nil, c.QueryAddr)
	if err != nil {
		log.Error(err)
		c.doError(err)
		return decoded
	}
	// write query
	n, err := conn.Write([]byte(query))
	if n != len([]byte(query)) {
		err = fmt.Errorf("Only wrote %v/%v bytes", n, len([]byte(query)))
		log.Error(err)
		c.doError(err)
		return decoded
	}
	decoder := json.NewDecoder(conn)
	decoder.UseNumber()
	err = decoder.Decode(&decoded)
	return decoded
}

// Subscribes to the given where clause (gets turned into "select * [where]") and returns
// results on the provided channel. The first item on the channel will be a giles.SmapMessageList,
// and the subsequent ones will be giles.SmapMessage until I finish implementing the diffs
func (c *Client) Subscribe(where string) (recv chan giles.QueryResult) {
	recv = make(chan giles.QueryResult)

	conn, err := net.DialTCP("tcp", nil, c.SubscribeAddr)
	if err != nil {
		log.Error(err)
		c.doError(err)
		return
	}
	// write where
	n, err := conn.Write([]byte(where))
	if n != len([]byte(where)) {
		err = fmt.Errorf("Only wrote %v/%v bytes", n, len([]byte(where)))
		log.Error(err)
		c.doError(err)
		return
	}

	go func() {
		reader := json.NewDecoder(conn)
		var msg giles.SmapMessage
		var msglist giles.SmapMessageList
		// initial message
		err = reader.Decode(&msglist)
		if err != nil {
			log.Error(err)
			c.doError(err)
			return
		}
		log.Debugf("send %v", msglist)
		recv <- msglist

		for reader.More() {
			err := reader.Decode(&msg)
			if err != nil {
				log.Error(err)
				c.doError(err)
				return
			}
			log.Debugf("send %v", msg)
			recv <- msg
		}
	}()
	return
}
