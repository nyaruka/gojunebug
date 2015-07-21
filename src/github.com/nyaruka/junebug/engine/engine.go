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
	Dispatcher     disp.Dispatcher
}

// Creates a new Connection object given the configuration and dispatcher, this is a factory
// method of sorts.
func NewConnectionEngine(conn *store.Connection) *ConnectionEngine {
	// create a dispatcher for this connection
	dispatcher := disp.CreateDispatcher(conn.NumSenders, conn.NumReceivers)

	// Create all our senders
	senders := make([]disp.MsgSender, conn.NumSenders)
	switch conn.SenderType {
	case "echo":
		for i := 0; i < conn.NumSenders; i++ {
			senders[i] = CreateEchoSender(i, conn, dispatcher.Senders, dispatcher.Incoming)
		}
	default:
		log.Fatal("Unsupported sender type: " + conn.SenderType)
	}

	// Then all our receivers
	receivers := make([]disp.MsgReceiver, conn.NumReceivers)
	switch conn.ReceiverType {
	case "http":
		for i := 0; i < conn.NumReceivers; i++ {
			receivers[i] = CreateHttpReceiver(i, conn, dispatcher.Receivers)
		}
	default:
		log.Fatal("Unsupported receiver type: " + conn.ReceiverType)
	}

	return &ConnectionEngine{
		Connection: conn,
		Senders: senders,
		Receivers: receivers,
		Dispatcher: dispatcher }
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
