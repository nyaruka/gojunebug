package store

import (
	"bytes"
	"encoding/json"
    "github.com/boltdb/bolt"
	"github.com/satori/go.uuid"
	"encoding/gob"
	"encoding/binary"
	"io"
	"errors"
	"time"
	"fmt"
	"sync"
)

type Msg struct {
	Id         uint64    `json:"id"`
	ConnUuid   string    `json:"conn_uuid"`
	Address    string    `json:"address"`
	Text       string    `json:"text"`
	Priority   string    `json:"priority"`
	Status     string    `json:"status"`
	Log        string    `json:"log"`
	Created    time.Time `json:"created"`
	Finished   time.Time `json:"finished"`
}

type Connection struct {
	Uuid               string `json:"uuid"`

	Senders struct {
		Type           string            `json:"type"`
		Count          uint              `json:"count"`
	    Config         map[string]string `json:"config"` } `json:"senders"`


	Receivers struct {
		Type           string            `json:"type"`
		Count          uint              `json:"count"`
	    Config         map[string]string `json:"config"` } `json:"receivers"`
}

type ConnectionStatus struct {
	OutgoingQueued int `json:"outgoing_queued"`
	IncomingQueued int `json:"incoming_queued"`
	HandledResults int `json:"handled_results"`
	SentResults    int `json:"sent_results"`
}

const OUTBOX_BUCKET = "outbox"
const SENT_BUCKET = "sent"
const INBOX_BUCKET = "inbox"
const HANDLED_BUCKET = "handled"
const MSG_BUCKET = "msgs"
const CONNECTION_BUCKET = "connections"

const STATUS_QUEUED = "Q"
const STATUS_SENT = "S"
const STATUS_HANDLED = "H"

const PRIORITY_HIGH = "H"
const PRIORITY_LOW = "L"

const LOW_PRIORITY_MASK = 1<<63

// our global DB connection
var db *bolt.DB

func OpenDB(filename string) (*bolt.DB, error) {
	var err error
    db, err = bolt.Open(filename, 0600, nil)

	// Create our connection bucket, so that our views can be read only
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(CONNECTION_BUCKET))
		return err
	})

	return db, err
}

func CloseDB() error {
	return db.Close()
}

//------------------------------------------------------------------------
// Msg Pool (this saves us GC allocations as we build a LOT of msgs)
// ------------------------------------------------------------------------
var msgPool = sync.Pool{
    New: func() interface{} {
        return &Msg{}
    },
}

//------------------------------------------------------------------------
// Bolt Operations
//------------------------------------------------------------------------

func getMsgBucket(tx *bolt.Tx, connection string, bucket string) (b *bolt.Bucket, err error) {
	// grab our connections bucket
	b = tx.Bucket([]byte(CONNECTION_BUCKET))
	if b == nil {
		return b, errors.New("Unable to get \"connections\" bucket")
	}

	// make sure our connection bucket exists
	b = tx.Bucket([]byte(connection))
	if b == nil {
		return b, errors.New(fmt.Sprintf("Unable to get connection bucket \"%s\"", connection))
	}

	// and make sure our sub bucket exists
	b = b.Bucket([]byte(bucket))
	if b == nil {
		return b, errors.New(fmt.Sprintf("Unable get bucket: \"%s\" for connection \"%s\"", connection, bucket))
	}

	return b, err
}

func ensureMsgBucket(tx *bolt.Tx, connection string, bucket string) (b *bolt.Bucket, err error) {
	// make sure our connection bucket exists
	b, err = tx.CreateBucketIfNotExists([]byte(connection))
	if err != nil {
		return b, err
	}

	// and make sure our sub bucket exists
	b, err = b.CreateBucketIfNotExists([]byte(bucket))
	if err != nil {
		return b, err
	}

	return b, err
}

func getMsgBucketKeys(connection string, bucket string) (*[]uint64, error) {
	var k *[]uint64
	return k, db.View(func(tx *bolt.Tx) error {
		b, err := getMsgBucket(tx, connection, bucket)
		if err != nil {
			return err
		}

		keys := make([]uint64, 0, 1000)

		// read all the keys in this bucket
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, binary.LittleEndian.Uint64(k))
		}

		// set our return pointer
		k = &keys

		return nil
	})
}

func saveMsgToBucket(msg *Msg, addBucket string, deleteBucket string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := getMsgBucket(tx, msg.ConnUuid, MSG_BUCKET)
		if err != nil {
			return err
		}

		// create an id if we don't have one
		if msg.Id == 0 {
			msg.Id, err = b.NextSequence()
			if err != nil {
				return err
			}

			// if we are low priority, use our bitmask to shift it behind all others
			if msg.Priority == PRIORITY_LOW {
				msg.Id |= LOW_PRIORITY_MASK
			}
		}

		// encode our msg using gob
		msgBuf := &bytes.Buffer{}
		enc := gob.NewEncoder(msgBuf)
		err = enc.Encode(msg)
		if err != nil {
			return err
		}

		// and encode our id
		idBuf := make([]byte, 8, 8)
		binary.LittleEndian.PutUint64(idBuf, msg.Id)

		// write our msg
		err = b.Put(idBuf, msgBuf.Bytes())
		if err != nil {
			return err
		}

		// if we have bucket to add to, insert there
		if addBucket != "" {
			b, err := getMsgBucket(tx, msg.ConnUuid, addBucket)
			if err != nil {
				return err
			}

			timeBuf := make([]byte, 8, 8)
			binary.LittleEndian.PutUint64(timeBuf, uint64(time.Now().UnixNano()))

			err = b.Put(idBuf, timeBuf)
			if err != nil {
				return err
			}
		}

		// if we have a bucket to remove from, delete there
		if deleteBucket != "" {
			b, err := getMsgBucket(tx, msg.ConnUuid, deleteBucket)
			if err != nil {
				return err
			}

			err = b.Delete(idBuf)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func getMsg(connection string, id uint64) (*Msg, error) {
	msg := msgPool.Get().(*Msg)
	msg.init()
	return msg, db.View(func(tx *bolt.Tx) error {
		b, err := getMsgBucket(tx, connection, MSG_BUCKET)
		if err != nil {
			return err
		}

		idBuf := make([]byte, 8, 8)
		binary.LittleEndian.PutUint64(idBuf, uint64(id))

		msgBytes := b.Get(idBuf)


		dec := gob.NewDecoder(bytes.NewReader(msgBytes))
		err = dec.Decode(msg)
		return err
	})
}

func deleteConnection(connection *Connection) (err error) {
	return db.Update(func(tx *bolt.Tx) error {
		// Delete our connection from the connnections bucket
		b := tx.Bucket([]byte(CONNECTION_BUCKET))

		// delete our config
		if b != nil {
			err = b.Delete([]byte(connection.Uuid))
		}

		// keep going even if we have an error, still have to remove our msg bucket
		b = tx.Bucket([]byte(connection.Uuid))

		if b != nil {
			return tx.DeleteBucket([]byte(connection.Uuid))
		} else {
			return err
		}
	})
}

func saveConnection(connection *Connection) error {
	return db.Update(func(tx *bolt.Tx) error {
		// Create our bucket
		b, err := tx.CreateBucketIfNotExists([]byte(CONNECTION_BUCKET))
		if err != nil {
			return err
		}

		// ensure all our buckets exist
		required_buckets := []string{OUTBOX_BUCKET, SENT_BUCKET, INBOX_BUCKET, HANDLED_BUCKET, MSG_BUCKET}
		for _, bucket_name := range(required_buckets) {
			_, err := ensureMsgBucket(tx, connection.Uuid, bucket_name)
			if err != nil {
				return err
			}
		}

		// encode our Connection using gob
		connBuf := &bytes.Buffer{}
		enc := gob.NewEncoder(connBuf)
		err = enc.Encode(connection)
		if err != nil {
			return err
		}

		// finally, write it to our db
		return b.Put([]byte(connection.Uuid), connBuf.Bytes())
	})
}

func loadConnection(uuid string) (*Connection, error) {
	var connection Connection
	return &connection, db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(CONNECTION_BUCKET))
		if b == nil {
			return errors.New(fmt.Sprintf("Unable to get connection bucket: %s", CONNECTION_BUCKET))
		}

		connBytes := b.Get([]byte(uuid))
		dec := gob.NewDecoder(bytes.NewReader(connBytes))
		err := dec.Decode(&connection)
		return err
	})
}

func getConnectionBucketSize(connUuid string, bucket string) (size int, err error){
	err = db.View(func(tx *bolt.Tx) error {
		b, err := getMsgBucket(tx, connUuid, bucket)
		if err != nil {
			return err
		}

		size = b.Stats().KeyN
		return nil
	})
	return size, err
}

//------------------------------------------------------------------------
// Msg Operations
//------------------------------------------------------------------------

// Write ourselves to the outbox
func (m *Msg) WriteToOutbox() (err error) {
	return saveMsgToBucket(m, OUTBOX_BUCKET, "")
}

// Write ourselves to the inbox
func (m *Msg) WriteToInbox() (err error) {
	return saveMsgToBucket(m, INBOX_BUCKET, "")
}

// Mark ourselves as sent, this just updates our status and saves
func (m *Msg) MarkSent(msgLog string) (err error) {
	m.Status = STATUS_SENT
	m.Finished = time.Now()
	m.Log = msgLog
	return saveMsgToBucket(m, SENT_BUCKET, OUTBOX_BUCKET)
}

// Mark ourselves as handled, this just update our status and saves
func (m *Msg) MarkHandled(msgLog string) (err error) {
	m.Status = STATUS_HANDLED
	m.Finished = time.Now()
	m.Log = msgLog
	return saveMsgToBucket(m, HANDLED_BUCKET, INBOX_BUCKET)
}

// Clears the values on this msg
func (m *Msg) init() {
	m.Id = 0
	m.ConnUuid = ""
	m.Address = ""
	m.Text = ""
	m.Priority = ""
	m.Status = ""
	m.Log = ""
	m.Created = time.Time{}
	m.Finished = time.Time{}
}

// Releases this message back to our pool
func (m *Msg) Release() {
	msgPool.Put(m)
}

// Reads an Inbox Msg
func MsgFromId(connUuid string, id uint64) (msg *Msg, err error) {
	return getMsg(connUuid, id)
}

// Builds a Msg object from the passed in text and from
func MsgFromText(connUuid string, from string, text string) *Msg {
	msg := msgPool.Get().(*Msg)
	msg.init()

	msg.ConnUuid = connUuid
	msg.Address = from
	msg.Text = text
	msg.Priority = PRIORITY_LOW
	msg.Status = STATUS_QUEUED
	msg.Created = time.Now()

	return msg
}

// Builds a Msg object from the passed in JSON
func MsgFromJson(body io.Reader) (*Msg, error) {
	msg := msgPool.Get().(*Msg)
	msg.init()

	// Decode it from the passed in JSON
	decoder := json.NewDecoder(body)
	err := decoder.Decode(msg)
	if err != nil {
		return msg, err
	}

	// to and text and required
	if msg.Address == "" || msg.Text == "" {
		return msg, errors.New("Must specify `address` and `text`")
	}

	if msg.Priority == "" {
		msg.Priority = PRIORITY_LOW
	}

	// check that priority is set correctly
	if msg.Priority != PRIORITY_HIGH && msg.Priority != PRIORITY_LOW {
		return msg, errors.New("`priority` must be one of `H` (high) or `L` (low)")
	}

	// all messages start as queued
	msg.Status = STATUS_QUEUED
	msg.Created = time.Now()

	// return our msg
	return msg, nil
}

//------------------------------------------------------------------------
// Connection Operations
//------------------------------------------------------------------------

// Writes this connection to disk
func (c *Connection) Save() (err error) {
	return saveConnection(c)
}

// Deletes this connection entirely, callers should make sure that
// any running ConnectionEngine's have been stopped beforehand
func (c *Connection) Delete() (err error) {
	return deleteConnection(c)
}

// loads our number of queued messages
func (c *Connection) GetStatus() (*ConnectionStatus, error) {
	var status ConnectionStatus
	var err error

	status.IncomingQueued, err = getConnectionBucketSize(c.Uuid, INBOX_BUCKET)
	if err != nil {
		return &status, err
	}

	status.OutgoingQueued, err = getConnectionBucketSize(c.Uuid, OUTBOX_BUCKET)
	if err != nil {
		return &status, err
	}

	status.HandledResults, err = getConnectionBucketSize(c.Uuid, HANDLED_BUCKET)
	if err != nil {
		return &status, err
	}

	status.SentResults, err = getConnectionBucketSize(c.Uuid, SENT_BUCKET)
	return &status, err
}

// loads the ids of our inbox messages
func (c *Connection) GetInboxMsgs() (ids *[]uint64, err error) {
	return getMsgBucketKeys(c.Uuid, INBOX_BUCKET)
}

// loads the ids of our outbox messages
func (c *Connection) GetOutboxMsgs() (ids *[]uint64, err error) {
	return getMsgBucketKeys(c.Uuid, OUTBOX_BUCKET)
}

// loads all our connections from the db
func LoadAllConnections() (*[]Connection, error) {
	var return_conns *[]Connection
	return return_conns, db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(CONNECTION_BUCKET))
		if b == nil {
			return errors.New(fmt.Sprintf("Unable to load connection bucket: %s", CONNECTION_BUCKET))
		}

		// default to something like 10
		connections := make([]Connection, 0, 10)

		// iterate across them
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var connection Connection
			dec := gob.NewDecoder(bytes.NewReader(v))
			err := dec.Decode(&connection)
			if err != nil {
				return err
			}

			connections = append(connections, connection)
		}

		// set our return value
		return_conns = &connections

		return nil
	})
}

// Builds a single configuration from a file
func ConnectionFromUuid(uuid string) (connection *Connection, err error) {
	return loadConnection(uuid)
}

// Builds a single configuration from JSON
func ConnectionFromJson(body io.Reader) (*Connection, error) {
	var connection Connection

	decoder := json.NewDecoder(body)
	err := decoder.Decode(&connection)
	if err != nil {
		return &connection, errors.New("Invalid JSON, please check the body of your request: " + err.Error())
	}

	// type is required
	if connection.Senders.Type == "" {
		return &connection, errors.New("Must specify a sender type in field `sender_type`")
	}

	if connection.Senders.Type != "echo" && connection.Senders.Type != "twitter" {
		return &connection, errors.New("Invalid sender_type, must be `echo` or `twitter`")
	}

	if connection.Receivers.Type == "" {
		connection.Receivers.Type = "http"
	}

	if connection.Receivers.Type != "http" {
		return &connection, errors.New("Invalid receiver_type, must be `http`")
	}

	if connection.Senders.Count == 0 {
		connection.Senders.Count = 1
	}

	if connection.Receivers.Count == 0 {
		connection.Receivers.Count = 1
	}

	// ok, all looks good, generate a new UUID
	connection.Uuid = uuid.NewV4().String()

	// and return it
	return &connection, nil
}