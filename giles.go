package main

import (
	"flag"
	"github.com/gtfierro/giles2/archiver"
	"github.com/op/go-logging"
	"os"
)

// config flags
var configfile = flag.String("c", "giles.cfg", "Path to Giles configuration file")

// logger
var log *logging.Logger

func init() {
	log = logging.MustGetLogger("giles")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

func main() {
	flag.Parse()
	config := archiver.LoadConfig(*configfile)
	archiver.PrintConfig(config)
}
