package http

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/nyaruka/junebug/cfg"
	"github.com/nyaruka/junebug/conn"
	"github.com/nyaruka/junebug/disp"
	"github.com/nyaruka/junebug/msg"
	"log"
	"net/http"
)

// our payload for a connection read response
type ReadConnectionResponse struct {
	Connection conn.ConnectionConfig `json:"connection"`
	Status     conn.ConnectionStatus `json:"status"`
}

// our payload for a msg read response
type ReadMsgResponse struct {
	Msg    msg.Msg       `json:"message"`
	Status msg.MsgStatus `json:"status"`
}

func addConnection(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// read the connection from the body
	config, err := conn.ConnectionConfigFromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// write the config to the filesystem
	configJson, err := config.Write(cfg.Config.Directories.Connections)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: this is both a terrible copy/paste hack and not safe, needs to be replaced
	dispatcher := disp.CreateDispatcher(config.NumSenders, config.NumReceivers)
	connection := conn.CreateConnection(config, dispatcher)

	// start everything
	dispatcher.Start()
	connection.Start()

	// assign it to our connection map so people can send on it
	connections[config.Uuid] = connection

	// write our config to the response
	w.Header().Set("Content-Type", "application/json")
	w.Write(configJson)
}

func listConnections(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	configs, err := conn.ReadConnectionConfigs(cfg.Config.Directories.Connections)
	connectionList := conn.ConnectionConfigList{configs}

	js, err := json.Marshal(connectionList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func readConnection(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uuid := ps.ByName("conn_uuid")

	var resp ReadConnectionResponse

	// load our connection config
	config, err := conn.ConnectionConfigFromUuid(cfg.Config.Directories.Connections, uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Connection = config

	// read our pending messages
	// TODO: replace with just count instead of reading everything in
	msgs, err := msg.ReadOutboxMsgs(uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Status.OutgoingQueued = len(msgs)

	// TODO: replace with just count instead of reading everything in
	msgs, err = msg.ReadInboxMsgs(uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Status.IncomingQueued = len(msgs)

	// output it
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	conn_uuid := ps.ByName("conn_uuid")

	// make sure this is a valid connection
	connection, exists := connections[conn_uuid]
	if !exists {
		http.Error(w, "No connection with uuid: "+conn_uuid, http.StatusBadRequest)
	}

	// read the message from our body
	msg, err := msg.MsgFromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// assign our connection UUID
	msg.ConnUuid = conn_uuid

	// write the msg to the filesystem
	msgJson, err := msg.WriteToOutbox()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// dispatch it
	connection.Dispatcher.Outgoing <- disp.MsgJob{msg.Uuid}

	// write our response
	w.Header().Set("Content-Type", "application/json")
	w.Write(msgJson)
}

func readMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	connUuid := ps.ByName("conn_uuid")
	msgUuid := ps.ByName("msg_uuid")

	var resp ReadMsgResponse

	// load our msg and status
	msg, status, err := msg.MsgFromUuid(connUuid, msgUuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Msg = msg
	resp.Status = status

	// output it
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

var connections map[string]conn.Connection

func StartServer(conns map[string]conn.Connection) {
	connections = conns

	router := httprouter.New()
	router.GET("/connection", listConnections)
	router.POST("/connection", addConnection)
	router.GET("/connection/:conn_uuid", readConnection)
	router.POST("/connection/:conn_uuid/send", sendMessage)
	router.GET("/connection/:conn_uuid/status/:msg_uuid", readMessage)

	log.Println(fmt.Sprintf("Starting server on http://localhost:%d", cfg.Config.Server.Port))
	log.Println("\tPOST /connection                      - Add a connection")
	log.Println("\tGET  /connection                      - List Connections")
	log.Println("\tGET  /connection/[uuid]               - Read Connection Status")
	log.Println("\tPOST /connection/[uuid]/send          - Send Message")
	log.Println("\tGET  /connection/[uuid]/status/[uuid] - Get Message Status")
	log.Println()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Config.Server.Port), router))
}
