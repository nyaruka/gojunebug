package conn

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "io"
  "io/ioutil"
  "encoding/json"
  "fmt"
  "path"
  "github.com/nyaruka/junebug/disp"
  "log"
)

// the configuration for a connection
type ConnectionConfig struct {
  Uuid string `json:"uuid"`
  SenderType string `json:"sender_type"`
  NumSenders int `json:"num_senders"`
  ReceiverType string `json:"receiver_type"`
  NumReceivers int `json:"num_receivers"`
  ReceiverUrl string `json:"receiver_url"`
}

// the status of a connection
type ConnectionStatus struct {
  OutgoingQueued int `json:"outgoing_queued"`
  IncomingQueued int `json:"incoming_queued"`
}

// slice of connections
type ConnectionConfigList struct {
  Connections []ConnectionConfig `json:"connections"`
}

// Writes this configuration to disk
func (c *ConnectionConfig) Write(directory string) (configJson []byte, err error){
  // re-encode as JSON
	js, err := json.Marshal(*c)
	if err != nil {
		return nil, err
	}

	// write our file out
	connPath := path.Join(directory, fmt.Sprintf("%s.json", c.Uuid))
	err = ioutil.WriteFile(connPath, js, 0644)
	if err != nil {
		return nil, err
	}

  return js, nil
}

// Reads all our connection configurations
func ReadConnectionConfigs(connectionDir string) (configs []ConnectionConfig, err error) {
  files, err := ioutil.ReadDir(connectionDir)
  if err != nil {
    return nil, err
  }

  connections := make([]ConnectionConfig, len(files))
  for i,file := range files {
    config, err := ConnectionConfigFromFile(path.Join(connectionDir, file.Name()))
    if err != nil {
        return nil, err
    }
    connections[i] = config
  }

  return connections, nil
}

// Builds a single configuration from a file
func ConnectionConfigFromFile(path string) (config ConnectionConfig, err error){
  fileContent, err := ioutil.ReadFile(path)
  if err != nil {
    return config, err
  }

  err = json.Unmarshal(fileContent, &config)
  if err != nil {
    return config, err
  }

  return config, nil
}

// Builds a single configuration given a connection uuid
func ConnectionConfigFromUuid(connDir string, uuid string) (ConnectionConfig, error) {
  return ConnectionConfigFromFile(path.Join(connDir, fmt.Sprintf("%s.json", uuid)))
}

// Builds a single configuration from JSON
func ConnectionConfigFromJson(body io.Reader) (config ConnectionConfig, err error){
  decoder := json.NewDecoder(body)
  err = decoder.Decode(&config)
  if err != nil {
    return config, err
  }

  // type is required
  if config.SenderType == "" {
    return config, errors.New("Must specify a sender type in field `sender_type`")
  }

  if config.SenderType != "echo" {
    return config, errors.New("Invalid sender_type, must be `echo`")
  }

  if config.ReceiverType == "" {
    config.ReceiverType = "http"
  }

  if config.ReceiverType != "http" {
    return config, errors.New("Invalid receiver_type, must be `http`")
  }

  if config.NumSenders == 0 {
    config.NumSenders = 1
  }

  if config.NumReceivers == 0 {
    config.NumReceivers = 1
  }

  if config.ReceiverUrl == "" {
    return config, errors.New("Must specify a receiver URL for incoming messages in field `receiver_url`")
  }

  // ok, all looks good, generate a new UUID for our connection and return it
  config.Uuid = uuid.New()
  return config, nil
}

type Connection struct {
  Config ConnectionConfig
  Senders []disp.MsgSender
  Receivers []disp.MsgReceiver
  Dispatcher disp.Dispatcher
}

// Creates a new Connection object given the configuration and dispatcher, this is a factory
// method of sorts.
func CreateConnection(config ConnectionConfig, dispatcher disp.Dispatcher) Connection {
    // Create all our senders
    senders := make([]disp.MsgSender, config.NumSenders)
    switch config.SenderType {
    case "echo":
      for i:=0; i < config.NumSenders; i++ {
        senders[i] = CreateEchoSender(i, config, dispatcher.Senders, dispatcher.Incoming)
      }
    default:
      log.Fatal("Unsupported sender type: " + config.SenderType)
    }

    // Then all our receivers
    receivers := make([]disp.MsgReceiver, config.NumReceivers)
    switch config.ReceiverType {
    case "http":
      for i:=0; i < config.NumReceivers; i++ {
        receivers[i] = CreateHttpReceiver(i, config, dispatcher.Receivers)
      }
    default:
      log.Fatal("Unsupported receiver type: " + config.ReceiverType)
    }

    return Connection{ Config:config, Senders:senders, Receivers:receivers, Dispatcher:dispatcher }
}

// Starts our connection and all it's senders as listening
func (c Connection) Start(){
  for _, sender := range(c.Senders) {
    sender.Start()
  }
  for _, receiver := range(c.Receivers) {
    receiver.Start()
  }
}
