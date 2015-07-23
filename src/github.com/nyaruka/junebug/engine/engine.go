package engine

import (
	"github.com/nyaruka/junebug/disp"
	"github.com/nyaruka/junebug/store"
	"log"
)

type ConnectionEngine struct {
	Connection     *store.Connection
	Senders        []disp.MsgSender
	Receivers      []disp.MsgReceiver
	Dispatcher     *disp.Dispatcher
}

// Creates a new Connection object given the configuration and dispatcher, this is a factory
// method of sorts.
func NewConnectionEngine(conn *store.Connection) (ce *ConnectionEngine, err error) {
	// create a dispatcher for this connection
	dispatcher := disp.CreateDispatcher(conn.Senders.Count, conn.Receivers.Count)

	// Create all our senders
	// TODO: refactor this so it isn't so repetitive
	senders := make([]disp.MsgSender, 0, conn.Senders.Count)
	switch conn.Senders.Type {
	case "echo":
		for i := 0; uint(i) < conn.Senders.Count; i++ {
			sender, err := CreateEchoSender(i, conn, dispatcher)
			if err != nil {
				return ce, err
			}
			senders = append(senders, sender)
		}
	case "twitter":
		for i := 0; uint(i) < conn.Senders.Count; i++ {
			sender, err := CreateTwitterConnection(i, conn, dispatcher)
			if err != nil {
				return ce, err
			}
			senders = append(senders, sender)
		}
	default:
		log.Fatal("Unsupported sender type: " + conn.Senders.Type)
	}

	// Then all our receivers
	receivers := make([]disp.MsgReceiver, 0, conn.Receivers.Count)
	switch conn.Receivers.Type {
	case "http":
		for i := 0; uint(i) < conn.Receivers.Count; i++ {
			receiver, err := CreateHttpReceiver(i, conn, dispatcher)
			if err != nil {
				return ce, err
			}
			receivers = append(receivers, receiver)
		}
	default:
		log.Fatal("Unsupported receiver type: " + conn.Receivers.Type)
	}

	return &ConnectionEngine{
		Connection: conn,
		Senders: senders,
		Receivers: receivers,
		Dispatcher: dispatcher }, err
}

func (c *ConnectionEngine) AddPendingMsgsFromDB() (outgoing int, incoming int, err error) {
	// dispatch any backlog of outgoing messages
	outgoing_ids, err := c.Connection.GetOutboxMsgs()
	if err != nil {
		return 0, 0, err
	}
	for _, id := range *outgoing_ids {
		c.Dispatcher.Outgoing <- id
	}

	// dispatch any backlog of messages
	incoming_ids, err := c.Connection.GetInboxMsgs()
	if err != nil {
		return 0, 0, err
	}
	for _, id := range *incoming_ids {
		c.Dispatcher.Incoming <- id
	}

	return len(*outgoing_ids), len(*incoming_ids), err
}

// Shuts down our connection
func (c *ConnectionEngine) Stop() {
	c.Dispatcher.Stop()
}

// Starts our connection and all it's senders as listening
func (c *ConnectionEngine) Start() {
	// Start our dispatcher
	c.Dispatcher.Start()

	// Our senders
	for _, sender := range c.Senders {
		sender.Start()
	}

	// And receivers
	for _, receiver := range c.Receivers {
		receiver.Start()
	}
}
