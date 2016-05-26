package archiver

import (
	"fmt"
	"gopkg.in/gcfg.v1"
)

type Config struct {
	Archiver struct {
		TimeseriesStore *string
		MetadataStore   *string
		Objects         *string
		Keepalive       *int
		EnforceKeys     bool
		LogLevel        *string
		MaxConnections  *int
		PeriodicReport  bool
	}

	ReadingDB struct {
		Port    *string
		Address *string
	}

	BtrDB struct {
		Port    *string
		Address *string
	}

	Quasar struct {
		Port    *string
		Address *string
	}

	Mongo struct {
		Port           *string
		Address        *string
		UpdateInterval *int
	}

	Venkman struct {
		Port    *string
		Address *string
	}

	HTTP struct {
		Enabled bool
		Port    *int
	}

	BOSSWAVE struct {
		Enabled    bool
		Entityfile string
		Namespace  string
	}

	WebSocket struct {
		Enabled bool
		Port    *int
	}

	MsgPackUDP struct {
		Enabled bool
		Port    *int
	}

	TCPJSON struct {
		Enabled       bool
		AddPort       *int
		QueryPort     *int
		SubscribePort *int
	}

	SSH struct {
		Enabled            bool
		Port               *string
		PrivateKey         *string
		AuthorizedKeysFile *string
		User               *string
		Pass               *string
		PasswordEnabled    bool
		KeyAuthEnabled     bool
	}

	Profile struct {
		CpuProfile     *string
		MemProfile     *string
		BenchmarkTimer *int
		Enabled        bool
	}
}

func LoadConfig(filename string) *Config {
	var configuration Config
	err := gcfg.ReadFileInto(&configuration, filename)
	if err != nil {
		log.Errorf("No configuration file found at %v, so checking current directory for giles.cfg (%v)", filename, err)
	} else {
		return &configuration
	}
	err = gcfg.ReadFileInto(&configuration, "./giles.cfg")
	if err != nil {
		log.Fatal("Could not find configuration files ./giles.cfg. Try retreiving a sample from github.com/gtfierro/giles")
	} else {
		return &configuration
	}
	return &configuration
}

func PrintConfig(c *Config) {
	fmt.Println("Giles Configuration")
	fmt.Println("Connecting to Mongo at", *c.Mongo.Address, ":", *c.Mongo.Port, "with update interval", *c.Mongo.UpdateInterval, "seconds")
	fmt.Println("Using Timeseries DB", *c.Archiver.TimeseriesStore)
	switch *c.Archiver.TimeseriesStore {
	case "readingdb":
		fmt.Println("	at address", *c.ReadingDB.Address, ":", *c.ReadingDB.Port)
	case "quasar":
		fmt.Println("	at address", *c.Quasar.Address, ":", *c.Quasar.Port)
	}
	fmt.Println("	with keepalive", *c.Archiver.Keepalive)

	if c.Profile.Enabled {
		fmt.Println("Profiling enabled for", *c.Profile.BenchmarkTimer, "seconds!")
		fmt.Println("CPU:", *c.Profile.CpuProfile)
		fmt.Println("Mem:", *c.Profile.MemProfile)
	} else {
		fmt.Println("Profiling disabled")
	}
}
