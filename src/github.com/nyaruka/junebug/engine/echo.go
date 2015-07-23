package engine

// I really want this to live in /senders but engine doesn't work then.. hrmm

import (
	"github.com/nyaruka/junebug/disp"
	"github.com/nyaruka/junebug/store"
	"log"
	"fmt"
	"sync"
	"time"
	"strconv"
	"errors"
)

// EchoSender is a dummy sender that takes 5 seconds to send anything, then returns an
// echo of the sent message back through our connection.
//
// It is an implementation of MsgSender
//

const PAUSE = "pause"

type EchoSender struct {
	id               int
	connection       store.Connection
	readySenders     chan disp.MsgSender
	pendingMsg       chan uint64
	incoming         chan uint64 // for receiving out our echos
	done             chan int
	wg               *sync.WaitGroup
	pause            uint
}

func (s EchoSender) Send(id uint64) {
	s.pendingMsg <- id
}

// Starts our sender, this starts a goroutine that blocks on receiving a message to send
func (s EchoSender) Start() {
	go func() {
		s.wg.Add(1)
		defer s.wg.Done()
		var id uint64

		for {
			// mark ourselves as ready for work, this never blocks
			s.readySenders <- s

			// wait for a job to come in, or for us to be shut down
			select {
			case id = <-s.pendingMsg:
			case <- s.done:
			  return
			}

			var msgLog = ""

			// load our msg
			msg, err := store.MsgFromId(s.connection.Uuid, id)
			if err != nil {
				msgLog = fmt.Sprintf("[%s][%d] Error sending msg (%d): %s", s.connection.Uuid, s.id, id, err.Error())
			} else {
				// sleep a bit to slow things down
				time.Sleep(time.Second * 5)
				msgLog = fmt.Sprintf("XXXX YYYY ZZZZ AAAA This is a log.\n" +
				                     "XXXX YYYY ZZZZ BBBB It is fake.")
			}

			// mark the message as sent
			err = msg.MarkSent(msgLog)
			if err != nil {
				log.Printf("[%s][%d] Error marking msg sent (%d)", s.connection.Uuid, s.id, id)
			} else {
				log.Printf("[%s][%d] Sent msg (%d)", s.connection.Uuid, s.id, id)
			}

			// release our message back to the pool
			msg.Release()

			// create a new incoming msg
			incoming := store.MsgFromText(s.connection.Uuid, msg.Address, "echo: "+msg.Text)
			err = incoming.WriteToInbox()
			if err != nil {
				log.Printf("[%s][%d] Error adding incoming msg (%d)", s.connection.Uuid, s.id, id)
			}

			// schedule it to go out
			s.incoming <- incoming.Id

			// release our msg back to the pool
			incoming.Release()
		}
	}()
}

func CreateEchoSender(id int, conn *store.Connection, dispatcher *disp.Dispatcher) (e *EchoSender, err error) {
	echo := EchoSender{
		id: id,
		connection:       *conn,
		readySenders:     dispatcher.Senders,
		incoming:         dispatcher.Incoming,
		pendingMsg:       make(chan uint64),
		done:             dispatcher.Done,
		wg:               dispatcher.WaitGroup }

	pause, _ := strconv.ParseInt(conn.Senders.Config[PAUSE], 10, 8)
	if pause < 0 {
		return e, errors.New(fmt.Sprintf("Pause must be 0 or a positive integer, was: %d", pause))
	}
	echo.pause = uint(pause)
	return &echo, err
}
