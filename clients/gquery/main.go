package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/gtfierro/giles2/clients/gquery/api"
	messages "github.com/gtfierro/giles2/plugins/bosswave"
	bw "gopkg.in/immesys/bw2bind.v5"
	"gopkg.in/readline.v1"
	"os"
	"os/user"
)

func doQuery(c *cli.Context) error {
	client := bw.ConnectOrExit("")
	vk := client.SetEntityFileOrExit(c.String("entity"))
	client.OverrideAutoChainTo(true)
	API := api.NewAPI(client, vk, c.String("archiver"))
	return API.Query(c.String("query"))
}

func doIQuery(c *cli.Context) error {
	client := bw.ConnectOrExit("")
	vk := client.SetEntityFileOrExit(c.String("entity"))
	client.OverrideAutoChainTo(true)
	API := api.NewAPI(client, vk, c.String("archiver"))

	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	completer := readline.NewPrefixCompleter(
		readline.PcItem("select",
			readline.PcItem("data",
				readline.PcItem("in"),
				readline.PcItem("before"),
				readline.PcItem("after"),
			),
			readline.PcItem("Metadata/"),
			readline.PcItem("distinct",
				readline.PcItem("Metadata/"),
				readline.PcItem("uuid/"),
			),
			readline.PcItem("uuid"),
		),
	)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "(gquery)>",
		AutoComplete: completer,
		HistoryFile:  currentUser.HomeDir + "/.gqueryhist",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			fmt.Println(err)
			break
		}
		API.Query(line)
	}
	return nil
}

func doSubscribe(c *cli.Context) error {
	client := bw.ConnectOrExit("")
	vk := client.SetEntityFileOrExit(c.String("entity"))
	client.OverrideAutoChainTo(true)
	API := api.NewAPI(client, vk, c.String("archiver"))
	return API.SubscribeData(c.String("query"), dump)
}

func dump(ts messages.QueryTimeseriesResult) {
	if len(ts.Data) > 0 {
		ts.Dump()
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "gquery"
	app.Version = "0.0.3"

	app.Commands = []cli.Command{
		{
			Name:   "query",
			Usage:  "Evaluate query",
			Action: doQuery,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "entity,e",
					Value:  "",
					Usage:  "The entity to use",
					EnvVar: "BW2_DEFAULT_ENTITY",
				},
				cli.StringFlag{
					Name:  "archiver,a",
					Value: "gabe.ns",
					Usage: "REQUIRED. The URI you want to archive",
				},
				cli.StringFlag{
					Name:  "query,q",
					Value: "",
					Usage: "Giles query string",
				},
			},
		},
		{
			Name:   "iquery",
			Usage:  "Evaluate query interactively",
			Action: doIQuery,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "entity,e",
					Value:  "",
					Usage:  "The entity to use",
					EnvVar: "BW2_DEFAULT_ENTITY",
				},
				cli.StringFlag{
					Name:  "archiver,a",
					Value: "gabe.ns",
					Usage: "REQUIRED. The URI you want to archive",
				},
			},
		},
		{
			Name:   "subscribe",
			Usage:  "Subscribe to Giles",
			Action: doSubscribe,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "entity,e",
					Value:  "",
					Usage:  "The entity to use",
					EnvVar: "BW2_DEFAULT_ENTITY",
				},
				cli.StringFlag{
					Name:  "archiver,a",
					Value: "gabe.ns",
					Usage: "REQUIRED. The URI you want to archive",
				},
				cli.StringFlag{
					Name:  "query,q",
					Value: "",
					Usage: "Giles query string",
				},
			},
		},
	}
	app.Run(os.Args)
}
