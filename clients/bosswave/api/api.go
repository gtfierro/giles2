package api

import (
	"fmt"
	messages "github.com/gtfierro/giles2/plugins/bosswave"
	bw "gopkg.in/immesys/bw2bind.v5"
	"math/rand"
	"strings"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type API struct {
	client *bw.BW2Client
	vk     string
	uri    string
}

// Create a new API isntance w/ the given client and VerifyingKey.
// The verifying key is returned by any of the BW2Client.SetEntity* calls
// URI should be the base of the giles service
func NewAPI(client *bw.BW2Client, vk string, uri string) *API {
	return &API{
		client: client,
		vk:     vk,
		uri:    strings.TrimSuffix(uri, "/") + "/s.giles/0/i.archiver",
	}
}

func (api *API) Query(query string) error {
	var wg sync.WaitGroup
	nonce := rand.Uint32()
	msg := messages.KeyValueQuery{
		Query: query,
		Nonce: nonce,
	}
	wg.Add(2)
	fmt.Printf("Subscribe to %v\n", api.uri+fmt.Sprintf("/signal/%s,queries", api.vk[:len(api.vk)-1]))
	c, err := api.client.Subscribe(&bw.SubscribeParams{
		URI: api.uri + fmt.Sprintf("/signal/%s,queries", api.vk[:len(api.vk)-1]),
	})
	if err != nil {
		return err
	}
	go func() {
		for msg := range c {
			if found, err := GetError(nonce, msg); found {
				fmt.Println(err)
				wg.Done()
				wg.Done()
			}
			if found, res, err := GetMetadata(nonce, msg); err == nil && found {
				fmt.Printf("Metadata %+v\n", res)
				wg.Done()
			} else if found && err != nil {
				fmt.Println(err)
				wg.Done()
			}
			if found, res, err := GetTimeseries(nonce, msg); err == nil && found {
				fmt.Printf("Timeseries %+v\n", res)
				wg.Done()
			} else if found && err != nil {
				fmt.Println(err)
				wg.Done()
			}
		}
	}()
	err = api.client.Publish(&bw.PublishParams{
		URI:            api.uri + "/slot/query",
		PayloadObjects: []bw.PayloadObject{msg.ToMsgPackBW()},
	})
	fmt.Printf("Publish to %v\n", api.uri+"/query")
	wg.Wait()
	return nil
}

// Extracts QueryError from Giles response. Returns false if no related message was found
func GetError(nonce uint32, msg *bw.SimpleMessage) (bool, error) {
	var (
		po         bw.PayloadObject
		queryError messages.QueryError
	)
	if po = msg.GetOnePODF(messages.GilesQueryErrorPIDString); po != nil {
		if err := po.(bw.MsgPackPayloadObject).ValueInto(&queryError); err != nil {
			return false, err
		}
		if queryError.Nonce != nonce {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// Extracts Metadata from Giles response. Returns false if no related message was found
func GetMetadata(nonce uint32, msg *bw.SimpleMessage) (bool, messages.QueryMetadataResult, error) {
	var (
		po              bw.PayloadObject
		metadataResults messages.QueryMetadataResult
	)
	if po = msg.GetOnePODF(messages.GilesQueryMetadataResultPIDString); po != nil {
		if err := po.(bw.MsgPackPayloadObject).ValueInto(&metadataResults); err != nil {
			return false, metadataResults, err
		}
		if metadataResults.Nonce != nonce {
			return false, metadataResults, nil
		}
		return true, metadataResults, nil
	}
	return false, metadataResults, nil
}

// Extracts Timeseries from Giles response. Returns false if no related message was found
func GetTimeseries(nonce uint32, msg *bw.SimpleMessage) (bool, messages.QueryTimeseriesResult, error) {
	var (
		po                bw.PayloadObject
		timeseriesResults messages.QueryTimeseriesResult
	)
	if po = msg.GetOnePODF(messages.GilesQueryTimeseriesResultPIDString); po != nil {
		if err := po.(bw.MsgPackPayloadObject).ValueInto(&timeseriesResults); err != nil {
			return false, timeseriesResults, err
		}
		if timeseriesResults.Nonce != nonce {
			return false, timeseriesResults, nil
		}
		return true, timeseriesResults, nil
	}
	return false, timeseriesResults, nil
}