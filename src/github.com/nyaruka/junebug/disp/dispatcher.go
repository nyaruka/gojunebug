package disp

import (
	"sort"
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

	available_outgoing []uint64
	available_senders  []MsgSender

	available_incoming  []uint64
	available_receivers []MsgReceiver
}

func CreateDispatcher(nsenders int, nreceivers int) Dispatcher {
	dispatcher := Dispatcher{Outgoing: make(chan uint64),
		Senders:   make(chan MsgSender, nsenders),
		Incoming:  make(chan uint64, nsenders),
		Receivers: make(chan MsgReceiver, nreceivers),
		available_outgoing:  make([]uint64, 0, 1000),
		available_senders:   make([]MsgSender, 0, nsenders),
		available_incoming:  make([]uint64, 0, 1000),
		available_receivers: make([]MsgReceiver, 0, nreceivers)}
	return dispatcher
}

func insertSorted(sorted *[]uint64, id uint64) (*[]uint64) {
	ids := *sorted
	i := sort.Search(len(ids), func(i int) bool { return (ids)[i] >= id })

	// special case inserting at the end, which is a simple append
	if i == len(ids){
		ids = append(ids, id)
	} else {
		// we are inserting in the middle somewhere, use slicing
		ids = append(ids, 0)
		copy(ids[i+1:], ids[i:])
		ids[i] = id
	}

	return &ids
}

// Starts our goroutine that will accept jobs and available senders
// and match them as they come in
func (d Dispatcher) Start() {
	go func() {
		for {
			select {
			case outgoing := <-d.Outgoing:
			    new_outgoing := insertSorted(&d.available_outgoing, outgoing)
			    d.available_outgoing = *new_outgoing
			case sender := <-d.Senders:
				d.available_senders = append(d.available_senders, sender)
			case incoming := <-d.Incoming:
			    new_incoming := insertSorted(&d.available_incoming, incoming)
				d.available_incoming = *new_incoming
			case receiver := <-d.Receivers:
				d.available_receivers = append(d.available_receivers, receiver)
			}

			// while we have possible pairings of outgoing messages
			for len(d.available_senders) > 0 && len(d.available_outgoing) > 0 {
				msg := d.available_outgoing[0]
				sender := d.available_senders[0]
				sender.Send(msg)

				// pop off the elements we just sent
				d.available_outgoing = d.available_outgoing[1:]
				d.available_senders = d.available_senders[1:]
			}

			// while we have possible pairings of incoming messages
			for len(d.available_receivers) > 0 && len(d.available_incoming) > 0 {
				msg := d.available_incoming[0]
				receiver := d.available_receivers[0]
				receiver.Receive(msg)

				// pop off the elements we just sent
				d.available_incoming = d.available_incoming[1:]
				d.available_receivers = d.available_receivers[1:]
			}
		}
	}()
}
