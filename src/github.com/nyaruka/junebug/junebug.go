package main

import (
	"flag"
	"fmt"
	"github.com/nyaruka/junebug/cfg"
	"github.com/nyaruka/junebug/engine"
	"github.com/nyaruka/junebug/http"
	"github.com/nyaruka/junebug/store"
	"log"
	"os"
	"runtime"
)

func main() {
	settings := flag.String("settings", "", "Our settings file")
	procs := flag.Int("procs", 4, "Max number of processors to use")
	flag.Parse()

	// they didn't pass in settings parameter, print some help
	if *settings == "" {
		fmt.Println("\nUsage: junebug --settings=junebug.conf\n")
		fmt.Println("Example configuration file:\n")
		fmt.Println(cfg.GetSampleConfig())
		fmt.Println()
		os.Exit(1)
	}

	config, err := cfg.ReadConfig(*settings)
	if err != nil {
		fmt.Println("Error reading Junebug settings:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	runtime.GOMAXPROCS(*procs)

	// Open our Database
	store.OpenDB(config.DB.Filename)

	// load our connection configurations
	connections, err := store.LoadAllConnections()
	if err != nil {
		log.Fatal(err)
	}

	// for each one, create a real connection
	engines := make(map[string]*engine.ConnectionEngine)
	for i := 0; i < len(*connections); i++ {
		connection := (*connections)[i]

		// and create our actual connection
		engine := engine.NewConnectionEngine(&connection)
		engine.Start()
		incoming, outgoing, err := engine.AddPendingMsgsFromDB()
		if err != nil {
			log.Fatal(err)
		}

		log.Println(
			fmt.Sprintf("[%s] Started with %d queued outgoing, %d queued incoming",
			connection.Uuid, outgoing, incoming))

		// stash it
		engines[connection.Uuid] = engine
	}

	// start our server
	http.StartServer(&engines)
}
