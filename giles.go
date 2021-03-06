package main

import (
	"flag"
	"github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/plugins/bosswave"
	"github.com/gtfierro/giles2/plugins/http"
	"github.com/gtfierro/giles2/plugins/msgpack"
	"github.com/gtfierro/giles2/plugins/tcpjson"
	"github.com/gtfierro/giles2/plugins/websocket"
	"github.com/op/go-logging"

	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"syscall"
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

	signals := make(chan os.Signal, 1)
	done := make(chan bool)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		log.Noticef("Got signal %v", sig)
		done <- true
	}()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			log.Infof("Number of active goroutines %v", runtime.NumGoroutine())
		}
	}()

	//time.AfterFunc(30*time.Second, func() {
	//	panic("STOP")
	//})

	flag.Parse()
	config := archiver.LoadConfig(*configfile)
	archiver.PrintConfig(config)

	/** Configure CPU profiling */
	if config.Profile.Enabled {
		log.Infof("Benchmarking for %v seconds\n", *config.Profile.BenchmarkTimer)
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

		f3, err := os.Create("trace.out")
		if err != nil {
			log.Fatal(err)
		}
		trace.Start(f3)
		defer runtime.SetBlockProfileRate(0)
		defer pprof.Lookup("block").WriteTo(f2, 1)
		defer pprof.StopCPUProfile()
	}

	a := archiver.NewArchiver(config)

	if config.HTTP.Enabled {
		go http.Handle(a, *config.HTTP.Port)
	}

	if config.BOSSWAVE.Enabled {
		go bosswave.Handle(a, &config.BOSSWAVE)
	}

	if config.WebSocket.Enabled {
		go websocket.Handle(a, *config.WebSocket.Port)
	}

	if config.MsgPackUDP.Enabled {
		go msgpack.HandleUDP4(a, *config.MsgPackUDP.Port)
	}

	if config.TCPJSON.Enabled {
		go tcpjson.Handle(a, *config.TCPJSON.AddPort, *config.TCPJSON.QueryPort, *config.TCPJSON.SubscribePort)
	}

	<-done
}
