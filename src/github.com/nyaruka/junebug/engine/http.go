package engine

// I really want this to live in /receiver but engine doesn't work then.. hrmm

import (
	"bytes"
	"encoding/json"
	"github.com/nyaruka/junebug/disp"
	"github.com/nyaruka/junebug/store"
	"log"
	"net/http"
	"fmt"
	"sync"
	"errors"
)

const RECEIVE_URL = "url"

// Http Receiver is a basic receiver that forwards the incoming message to an endpoint
type HttpReceiver struct {
	id               int
	connection       store.Connection
	readyReceivers   chan disp.MsgReceiver
	pendingMsg       chan uint64
	done             chan int
	wg               *sync.WaitGroup
	url              string
}

func (s HttpReceiver) Receive(id uint64) {
	s.pendingMsg <- id
}

// Starts our receiver, this starts a goroutine that blocks on msgs to forward
func (r HttpReceiver) Start() {
	go func() {
		// tell our wait group we started
		r.wg.Add(1)

		// when we exit, tell our wait group we stopped
		defer r.wg.Done()
		var id uint64

		for {
			// mark ourselves as ready for work, this never blocks
			r.readyReceivers <- r

			// wait for a job to come in, or be marked as complete
			select {
			case id = <-r.pendingMsg:
			case <-r.done: return
			}

			// load our msg
			var msgLog = ""
			msg, err := store.MsgFromId(r.connection.Uuid, id)
			if err != nil {
				msgLog = fmt.Sprintf(
					"[%s][%d] Error loading msg (%d) from store: %s", r.connection.Uuid, r.id, id, err.Error())
			} else {
				js, err := json.Marshal(msg)
				if err != nil {
					msgLog = fmt.Sprintf(
						"[%s][%d] Error json encoding msg (%d): %s", r.connection.Uuid, r.id, id, err.Error())
				} else {
					// we post our Msg body to our receiver URL
					req, err := http.NewRequest("POST", r.url, bytes.NewBuffer(js))
					req.Header.Set("Content-Type", "application/json")

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						msgLog = fmt.Sprintf("[%s][%d] Error posting msg (%d): %s", r.connection.Uuid, r.id, id, err.Error())
					} else {
						buf := new(bytes.Buffer)
						buf.ReadFrom(resp.Body)
						body := buf.String()

						if resp.Status != "200" && resp.Status != "201" {
							msgLog = fmt.Sprintf("[%s][%d] Error posting msg (%d) received status %s: %s",
								r.connection.Uuid, r.id, id, resp.Status, body)
						} else {
							msgLog = fmt.Sprintf("Status: %s\n\n%s", resp.Status, body)
						}
						resp.Body.Close()
					}
				}
			}

			// mark the message as sent
			err = msg.MarkHandled(msgLog)
			log.Printf("[%s][%d] Handled msg (%d)", r.connection.Uuid, r.id, id)
			if err != nil {
				log.Println("Error marking msg handled")
			}

			// release our msg back to our object pool
			msg.Release()
		}
	}()
}

func CreateHttpReceiver(id int, conn *store.Connection, dispatcher *disp.Dispatcher) (r *HttpReceiver, err error) {
	receiver := HttpReceiver{
		id: id,
		connection:       *conn,
		readyReceivers:   dispatcher.Receivers,
		pendingMsg:       make(chan uint64),
		done:             dispatcher.Done,
		wg:               dispatcher.WaitGroup }

	receiver.url = conn.Receivers.Config[RECEIVE_URL]
	if receiver.url == "" {
		return r, errors.New("You must specify a `url` in your configuration")
	}

	return &receiver, err
}
