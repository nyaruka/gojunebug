package engine

import (
	"github.com/nyaruka/junebug/disp"
	"github.com/nyaruka/junebug/store"
	"github.com/nyaruka/junebug/cfg"
	"github.com/ChimeraCoder/anaconda"
	"log"
	"fmt"
	"sync"
	"errors"
	"net/url"
)

const ACCESS_TOKEN = "access_token"
const ACCESS_TOKEN_SECRET = "access_token_secret"

type TwitterConnection struct {
	id               int
	connection       store.Connection
	readySenders     chan disp.MsgSender
	pendingMsg       chan uint64
	incoming         chan uint64 // for receiving out our echos
	done             chan int
	wg               *sync.WaitGroup

	token            string
	secret           string
}

func (t TwitterConnection) Send(id uint64) {
	t.pendingMsg <- id
}

// Starts our sender, this starts a goroutine that blocks on receiving a message to send
func (t TwitterConnection) Start() {
	// todo: this doesn't need to happen for each connection
	anaconda.SetConsumerKey(cfg.Config.Twitter.Consumer_Key)
	anaconda.SetConsumerSecret(cfg.Config.Twitter.Consumer_Secret)

	// this is our sending thread
	go func() {
		t.wg.Add(1)
		defer t.wg.Done()
		var id uint64

		// configure our API
		api := anaconda.NewTwitterApi(t.token, t.secret)
		defer api.Close()

		for {
			// mark ourselves as ready for work, this never blocks
			t.readySenders <- t

			// wait for a job to come in, or for us to be shut down
			select {
			case id = <-t.pendingMsg:
			case <- t.done:
				return
			}

			var msgLog = ""

			// load our msg
			msg, err := store.MsgFromId(t.connection.Uuid, id)
			if err != nil {
				msgLog = fmt.Sprintf("[%s][%d] Error sending msg (%d): %s", t.connection.Uuid, t.id, id, err.Error())
			} else {
				// send the message
				dm, err := api.PostDMToScreenName(msg.Text, msg.Address)
				if err != nil {
					msgLog = fmt.Sprintf("[%s][%d] Error sending msg (%d): %s", t.connection.Uuid, t.id, id, err.Error())
				} else {
					msgLog = fmt.Sprintf("[%s][%d] Sent DM, id: %d", t.connection.Uuid, t.id, dm.Id)
				}
			}

			// mark the message as sent
			err = msg.MarkSent(msgLog)
			if err != nil {
				log.Printf("[%s][%d] Error marking msg sent (%d)", t.connection.Uuid, t.id, id)
			} else {
				log.Printf("[%s][%d] Sent msg (%d)", t.connection.Uuid, t.id, id)
			}

			// release our message back to the pool
			msg.Release()
		}
	}()

	// this is our receiving thread
	go func() {
		t.wg.Add(1)
		defer t.wg.Done()

		api := anaconda.NewTwitterApi(t.token, t.secret)
		defer api.Close()

		userStream := api.UserStream(url.Values{})
		var event interface{}

		for {
			// wait for a twitter event to arrive
			select {
			case event = <-userStream.C:
			case <- t.done:
				userStream.Interrupt()
				return
			}

			// see if this is a direct message
			dm, ok := event.(anaconda.DirectMessage)
			if !ok {
				continue
			}

			log.Printf("[%s][%d] Received DM from %s: %s", t.connection.Uuid, t.id, dm.SenderScreenName, dm.Text)

			// create a new msg from our DM
			msg := store.MsgFromText(t.connection.Uuid, dm.SenderScreenName, dm.Text)
			err := msg.WriteToInbox()
			if err != nil {
				log.Printf("[%s][%d] Error saving msg (%d): %s", t.connection.Uuid, t.id, dm.Id, err.Error())
			}

			// pass our message to be received
			t.incoming <- msg.Id

			// release our message back to the pool
			msg.Release()
		}
	}()
}

func CreateTwitterConnection(id int, conn *store.Connection, dispatcher *disp.Dispatcher) (t *TwitterConnection, err error) {
	twitter := TwitterConnection{
		id: id,
		connection:       *conn,
		readySenders:     dispatcher.Senders,
		incoming:         dispatcher.Incoming,
		pendingMsg:       make(chan uint64),
		done:             dispatcher.Done,
		wg:               dispatcher.WaitGroup }


	twitter.token = conn.Senders.Config[ACCESS_TOKEN]
	if twitter.token == "" {
		return t, errors.New("Missing required config field `access_token`")
	}

	twitter.secret = conn.Senders.Config[ACCESS_TOKEN_SECRET]
	if twitter.secret == "" {
		return t, errors.New("Missing required config field `access_token_secret`")
	}

	return &twitter, err
}
