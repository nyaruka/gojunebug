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
)

// Http Receiver is a basic receiver that forwards the incoming message to an endpoint
type HttpReceiver struct {
	id               int
	connection       store.Connection
	readyReceivers   chan disp.MsgReceiver
	pendingMsg       chan uint64
}

func (s HttpReceiver) Receive(id uint64) {
	s.pendingMsg <- id
}

// Starts our receiver, this starts a goroutine that blocks on msgs to forward
func (r HttpReceiver) Start() {
	go func() {
		for {
			// mark ourselves as ready for work
			r.readyReceivers <- r

			// wait for a job to come in
			id := <-r.pendingMsg

			// load our msg
			var msgLog = ""
			msg, err := store.MsgFromId(r.connection.Uuid, id)
			if err != nil {
				msgLog = fmt.Sprintf("[%s][%d] Error loading msg (%d) from store: ", r.connection.Uuid, r.id, id, err.Error())
			} else {
				js, err := json.Marshal(msg)
				if err != nil {
					msgLog = fmt.Sprintf("[%s][%d] Error json encoding msg (%d): ", r.connection.Uuid, r.id, id, err.Error())
				} else {
					// we post our Msg body to our receiver URL
					req, err := http.NewRequest("POST", r.connection.ReceiverUrl, bytes.NewBuffer(js))
					req.Header.Set("Content-Type", "application/json")

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						msgLog = fmt.Sprintf("[%s][%d] Error posting msg (%d): ", r.connection.Uuid, r.id, id, err.Error())
					} else {
						buf := new(bytes.Buffer)
						buf.ReadFrom(resp.Body)
						body := buf.String()

						if resp.Status != "200" || resp.Status != "201" {
							msgLog = fmt.Sprintf("[%s][%d] Error posting msg (%d) received status %d: ",
								r.connection.Uuid,
								r.id,
								id,
								body)
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
		}
	}()
}

func CreateHttpReceiver(id int, conn *store.Connection, readyReceivers chan disp.MsgReceiver) HttpReceiver {
	return HttpReceiver{id: id,
		connection:       *conn,
		readyReceivers:   readyReceivers,
		pendingMsg:       make(chan uint64)}
}
