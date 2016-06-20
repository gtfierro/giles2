package main

import (
	"github.com/codegangsta/cli"
	"github.com/gtfierro/giles2/clients/bosswave/api"
	bw "gopkg.in/immesys/bw2bind.v5"
	"os"
)

func doQuery(c *cli.Context) error {
	client := bw.ConnectOrExit("")
	vk := client.SetEntityFileOrExit(c.String("entity"))
	client.OverrideAutoChainTo(true)
	API := api.NewAPI(client, vk, c.String("archiver"))
	return API.Query(c.String("query"))
}

func main() {
	app := cli.NewApp()
	app.Name = "savepoint"
	app.Version = "0.0.1"

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
	}
	app.Run(os.Args)
}
