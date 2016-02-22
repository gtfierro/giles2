package main

import (
	"encoding/json"
	"github.com/fatih/color"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/op/go-logging"
	"github.com/parnurzeal/gorequest"
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
	Addr          string
	Subscriptions map[string]chan giles.SmapMessage
}

func NewClient(addr string) *Client {
	c := &Client{
		Addr:          addr,
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

func (c *Client) DecodeMessage(r gorequest.Response) (decoded *giles.SmapMessageList, err error) {
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	err = decoder.Decode(&decoded)
	return
}

func (c *Client) Query(query string) giles.SmapMessageList {
	request := gorequest.New()
	resp, _, errs := request.Post(c.Addr + "/api/query").Type("text").Send(query).End()
	if errs != nil {
		c.doError(errs...)
		return nil
	}
	messages, err := c.DecodeMessage(resp)
	if err != nil {
		c.doError(err)
		return nil
	}
	return *messages
}

func (c *Client) Subscribe(where string) (recv chan giles.SmapMessage) {
	recv = make(chan giles.SmapMessage)
	request := gorequest.New()
	resp, _, errs := request.Post(c.Addr + "/republish").Type("text").Send(where).End()
	if errs != nil {
		c.doError(errs...)
		return
	}
	b := make([]byte, 2048)
	for {
		n, err := resp.Body.Read(b)
		if err != nil {
			c.doError(err)
			return
		}
		log.Debugf("read %v bytes", n)
		log.Debugf(string(b))
	}
	return
}

func main() {
	c := NewClient("http://localhost:8079")
	messages := c.Query("select *;")
	log.Debugf("got %v messages", len(messages))
	c.Subscribe("has uuid")

}
