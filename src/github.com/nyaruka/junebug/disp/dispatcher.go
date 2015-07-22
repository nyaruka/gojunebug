package disp

import (
	"github.com/nyaruka/junebug/store"
)

// The dispatcher essentially acts a router between available senders for a connection
// and incoming msgs that need to be sent. goroutines communicate to it using the
// Jobs and Senders channels.
//
// As new messages in Junebug come in to be sent, they are sent to the Jobs channel, where
// they are popped off and queued.
//
// As senders free up, they are added to the available senders queue.
//
// The dispatcher takes care of matchine one with the other.
type MsgSender interface {
	Send(uint64)
	Start()
}

type MsgReceiver interface {
	Receive(uint64)
	Start()
}

type Dispatcher struct {
	Outgoing  chan uint64
	Senders   chan MsgSender
	Incoming  chan uint64
	Receivers chan MsgReceiver

	available_outgoing store.PriorityQueue
	available_senders  []MsgSender

	available_incoming  store.PriorityQueue
	available_receivers []MsgReceiver
}

func CreateDispatcher(nsenders int, nreceivers int) Dispatcher {
	dispatcher := Dispatcher{Outgoing: make(chan uint64),
		Senders:   make(chan MsgSender, nsenders),
		Incoming:  make(chan uint64, nsenders),
		Receivers: make(chan MsgReceiver, nreceivers),

		available_senders:   make([]MsgSender, 0, nsenders),
		available_receivers: make([]MsgReceiver, 0, nreceivers)}

	return dispatcher
}

// Starts our goroutine that will accept jobs and available senders
// and match them as they come in
func (d Dispatcher) Start() {
	go func() {
		for {
			select {
			case outgoing := <-d.Outgoing:
			    d.available_outgoing.Insert(outgoing)
			case sender := <-d.Senders:
				d.available_senders = append(d.available_senders, sender)
			case incoming := <-d.Incoming:
			    d.available_incoming.Insert(incoming)
			case receiver := <-d.Receivers:
				d.available_receivers = append(d.available_receivers, receiver)
			}

			// while we have possible pairings of outgoing messages
			for len(d.available_senders) > 0 && d.available_outgoing.Len() > 0 {
				msg := d.available_outgoing.Pop()
				sender := d.available_senders[0]
				sender.Send(msg)

				// pop off the elements we just sent
				d.available_senders = d.available_senders[1:]
			}

			// while we have possible pairings of incoming messages
			for len(d.available_receivers) > 0 && d.available_incoming.Len() > 0 {
				msg := d.available_incoming.Pop()
				receiver := d.available_receivers[0]
				receiver.Receive(msg)

				// pop off the elements we just sent
				d.available_receivers = d.available_receivers[1:]
			}
		}
	}()
}
