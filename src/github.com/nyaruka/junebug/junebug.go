package main

import (
  "os"
	"log"
  "fmt"
  "flag"
  "github.com/nyaruka/junebug/cfg"
  "github.com/nyaruka/junebug/http"
  "github.com/nyaruka/junebug/conn"
  "github.com/nyaruka/junebug/msg"
  "github.com/nyaruka/junebug/disp"
)

func main() {
    settings := flag.String("settings", "", "Our settings file")
    flag.Parse()

    // they didn't pass in settings parameter, print some help
    if *settings == "" {
      fmt.Println("\nUsage: junebug --settings=junebug.conf\n")
      fmt.Println("Example configuration file:\n")
      fmt.Println(cfg.GetSampleConfig())
      fmt.Println()
      os.Exit(1)
    }

    _, err := cfg.ReadConfig(*settings)
    if err != nil {
      fmt.Println("Error reading Junebug settings:")
      fmt.Println(err.Error())
      os.Exit(1)
    }

    // load our connection configurations
    configs, err := conn.ReadConnectionConfigs(cfg.Config.Directories.Connections)
    if err != nil {
      log.Fatal(err)
    }

    // for each one, create a real connection
    // TODO: this whole block belongs somewhere else
    connections := make(map[string]conn.Connection)
    for i:=0; i<len(configs); i++ {
      config := configs[i]

      // create a dispatcher for this connection
      dispatcher := disp.CreateDispatcher(config.NumSenders, config.NumReceivers)

      // and create our actual connection
      connection := conn.CreateConnection(config, dispatcher)

      // start everything
      dispatcher.Start()
      connection.Start()

      // dispatch any backlog of outgoing messages
      outgoing, err := msg.ReadOutboxMsgs(config.Uuid)
      if err != nil {
        log.Fatal(err)
      }
      for _, msg := range outgoing {
        connection.Dispatcher.Outgoing <- disp.MsgJob{msg.Uuid}
      }

      // dispatch any backlog of messages
      incoming, err := msg.ReadInboxMsgs(config.Uuid)
      if err != nil {
        log.Fatal(err)
      }
      for _, msg := range incoming {
        connection.Dispatcher.Incoming <- disp.MsgJob{msg.Uuid}
      }

      log.Println(fmt.Sprintf("[%s] Started with %d queued outgoing, %d queued incoming",
                  config.Uuid, len(outgoing), len(incoming)))

      // stash it
      connections[config.Uuid] = connection
    }

    // start our server
    http.StartServer(connections)
}
