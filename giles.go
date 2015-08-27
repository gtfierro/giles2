package main

import (
	"flag"
	"github.com/gtfierro/giles2/archiver"
)

// config flags
var configfile = flag.String("c", "giles.cfg", "Path to Giles configuration file")

func main() {
	flag.Parse()
	config := archiver.LoadConfig(*configfile)
	archiver.PrintConfig(config)
}
