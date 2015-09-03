package main

import (
	"flag"
	"github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/http"
	"github.com/op/go-logging"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
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

	/** Configure CPU profiling */
	if config.Profile.Enabled {
		log.Info("Benchmarking for %v seconds\n", *config.Profile.BenchmarkTimer)
		f, err := os.Create(*config.Profile.CpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		f2, err := os.Create("blockprofile.db")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		runtime.SetBlockProfileRate(1)
		defer runtime.SetBlockProfileRate(0)
		defer pprof.Lookup("block").WriteTo(f2, 1)
		defer pprof.StopCPUProfile()
	}

	a := archiver.NewArchiver(config)

	if config.HTTP.Enabled {
		go http.Handle(a, *config.HTTP.Port)
	}

	idx := 0
	for {
		time.Sleep(5 * time.Second)
		idx += 5
		if config.Profile.Enabled && idx == *config.Profile.BenchmarkTimer {
			if *config.Profile.MemProfile != "" {
				f, err := os.Create(*config.Profile.MemProfile)
				if err != nil {
					log.Panic(err)
				}
				pprof.WriteHeapProfile(f)
				f.Close()
				return
			}
			if *config.Profile.CpuProfile != "" {
				return
			}
		}
	}
}
