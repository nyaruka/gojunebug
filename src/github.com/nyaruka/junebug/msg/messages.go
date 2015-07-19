package msg

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "io"
  "io/ioutil"
  "encoding/json"
  "fmt"
  "path"
  "os"
  "github.com/nyaruka/junebug/cfg"
)

type Msg struct {
  Uuid	string `json:"uuid"`
  ConnUuid string `json:"conn_uuid"`
  Address	string `json:"address"`
  Text string `json:"text"`
}

type MsgStatus struct {
  Status string `json:"status"`
}

func (m Msg) WriteToDir(dir string) (msgJson []byte, err error) {
  // make sure our directory exists
  err = os.MkdirAll(dir, 0770)
  if err != nil {
    return nil, err
  }

  // re-encode as JSON
	js, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	// write our file out
	msgPath := path.Join(dir, fmt.Sprintf("%s.json", m.Uuid))
	err = ioutil.WriteFile(msgPath, js, 0644)
	if err != nil {
		return nil, err
	}

  return js, nil
}

// Write ourselves to the outbox for our connection.
// returns our JSON representation
func (m Msg) WriteToOutbox() (msgJson []byte, err error) {
  return m.WriteToDir(path.Join(cfg.Config.Directories.Outbox, m.ConnUuid))
}

// Write ourselves to the outbox for our connection.
// returns our JSON representation
func (m Msg) WriteToInbox() (msgJson []byte, err error) {
  return m.WriteToDir(path.Join(cfg.Config.Directories.Inbox, m.ConnUuid))
}

// Mark ourselves as sent, this just moves us from the outbox folder
// to the sent folder atomically
func (m Msg) MarkSent() (err error) {
  outboxFilename := path.Join(cfg.Config.Directories.Outbox, m.ConnUuid, fmt.Sprintf("%s.json", m.Uuid))

  // make sure our directory exists
  sentDir := path.Join(cfg.Config.Directories.Sent, m.ConnUuid)
  err = os.MkdirAll(sentDir, 0770)
  if err != nil {
    return err
  }

  sentFilename := path.Join(sentDir, fmt.Sprintf("%s.json", m.Uuid))

  err = os.Rename(outboxFilename, sentFilename)
  return err
}

// Mark ourselves as handled, this just moves us from the inbox folder
// to the handled folder atomically
func (m Msg) MarkHandled() (err error) {
  inboxFilename := path.Join(cfg.Config.Directories.Inbox, m.ConnUuid, fmt.Sprintf("%s.json", m.Uuid))

  // make sure our directory exists
  handledDir := path.Join(cfg.Config.Directories.Handled, m.ConnUuid)
  err = os.MkdirAll(handledDir, 0770)
  if err != nil {
    return err
  }

  handledFilename := path.Join(handledDir, fmt.Sprintf("%s.json", m.Uuid))

  err = os.Rename(inboxFilename, handledFilename)
  return err
}

// Reads all the msgs in the outbox for the passed in connection
func ReadOutboxMsgs(connUuid string) (msgs []Msg, err error) {
  return ReadMsgsInDir(path.Join(cfg.Config.Directories.Outbox, connUuid))
}

func ReadInboxMsgs(connUuid string) (msgs []Msg, err error) {
  return ReadMsgsInDir(path.Join(cfg.Config.Directories.Inbox, connUuid))
}

// Reads all the msgs in the passed in folder
func ReadMsgsInDir(dir string) (msgs []Msg, err error) {
  files, err := ioutil.ReadDir(dir)
  if err != nil {
    msgs = make([]Msg, 0)
    return msgs, nil
  }

  msgs = make([]Msg, len(files))
  for i,file := range files {
    config, err := MsgFromFile(path.Join(dir, file.Name()))
    if err != nil {
        return nil, err
    }
    msgs[i] = config
  }

  return msgs, nil
}

// Reads a msg from sent directory for the passed in connection and msg id
func MsgFromSent(connUuid string, msgUuid string) (msg Msg, err error){
  msgFilename := path.Join(cfg.Config.Directories.Sent, connUuid, fmt.Sprintf("%s.json", msgUuid))
  return MsgFromFile(msgFilename)
}

// Reads a msg from outbox directory for the passed in connection and msg id
func MsgFromOutbox(connUuid string, msgUuid string) (msg Msg, err error){
  msgFilename := path.Join(cfg.Config.Directories.Outbox, connUuid, fmt.Sprintf("%s.json", msgUuid))
  return MsgFromFile(msgFilename)
}

// Reads a msg from inbox directory for the passed in connection and msg id
func MsgFromInbox(connUuid string, msgUuid string) (msg Msg, err error){
  msgFilename := path.Join(cfg.Config.Directories.Inbox, connUuid, fmt.Sprintf("%s.json", msgUuid))
  return MsgFromFile(msgFilename)
}

// Find the msg with the connection id and msg id, will first try outbox, then sent
func MsgFromUuid(connUuid string, msgUuid string) (msg Msg, status MsgStatus, err error){
  // first check our outbox
  msg, err = MsgFromOutbox(connUuid, msgUuid)
  status = MsgStatus{}

  // found it, return it and our status
  if err == nil {
    status.Status = "queued"
    return msg, status, nil
  }

  // try from our sent
  msg, err = MsgFromSent(connUuid, msgUuid)

  // found it, return it and our status
  if err == nil {
    status.Status = "sent"
    return msg, status, nil
  }

  // we don't know about this message
  return msg, status, errors.New("No msg found for: " + msgUuid)
}

// Builds a Msg object from the passed in filename
func MsgFromFile(path string) (msg Msg, err error){
  fileContent, err := ioutil.ReadFile(path)
  if err != nil {
    return msg, err
  }

  err = json.Unmarshal(fileContent, &msg)
  if err != nil {
    return msg, err
  }

  return msg, nil
}

// Builds a Msg object from the passed in text and from
func MsgFromText(connUuid string, from string, text string) Msg {
  return Msg{ ConnUuid: connUuid,
              Address: from,
              Text: text,
              Uuid: uuid.New() }
}

// Builds a Msg object from the passed in JSON
func MsgFromJson(body io.Reader) (msg Msg, err error){
  decoder := json.NewDecoder(body)
  err = decoder.Decode(&msg)
  if err != nil {
    return msg, err
  }

  // to and text and required
  if msg.Address == "" || msg.Text == "" {
    return msg, errors.New("Must specify `address` and `text`")
  }

  // generate a new UUID and return it
  msg.Uuid = uuid.New()
  return msg, nil
}
