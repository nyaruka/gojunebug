package http

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/nyaruka/junebug/cfg"
	"github.com/nyaruka/junebug/store"
	"github.com/nyaruka/junebug/engine"
	"log"
	"net/http"
	"strconv"
)

// our payload for a connection read response
type ConnectionResponse struct {
	Connection *store.Connection       `json:"connection"`
	Status     *store.ConnectionStatus `json:"status"`
}

type ConnectionListResponse struct {
	Connection *[]store.Connection `json:"connections"`
}

func addConnection(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// read the connection from the body
	connection, err := store.ConnectionFromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// gentlemen, start your engines!
	engine, err := engine.NewConnectionEngine(connection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ok, things look good, let's start our connection
	err = connection.Save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// start our engines!
	engine.Start()
	engines[connection.Uuid] = engine

	// write our config to the response
	w.Header().Set("Content-Type", "application/json")

	// serialize to json
	js, err := json.Marshal(connection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func deleteConnection(w http.ResponseWriter, r *http.Request, ps httprouter.Params){
	uuid := ps.ByName("conn_uuid")
	var resp ConnectionResponse

	// load our connection config
	connection, err := store.ConnectionFromUuid(uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Connection = connection

	// load our status
	status, err := connection.GetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Status = status

	// output it
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// shut down our connection
	engine, exists := engines[uuid]
	if exists {
		engine.Stop()
		delete(engines, uuid)
	}

	// remove all our data for it
	connection.Delete()

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func listConnections(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	connections, err := store.LoadAllConnections()
	connectionList := ConnectionListResponse{connections}

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

	var resp ConnectionResponse

	// load our connection config
	connection, err := store.ConnectionFromUuid(uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Connection = connection

	// load our status
	status, err := connection.GetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

func sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	conn_uuid := ps.ByName("conn_uuid")

	// make sure this is a valid connection
	engine, exists := engines[conn_uuid]
	if !exists {
		http.Error(w, "No connection with uuid: "+conn_uuid, http.StatusBadRequest)
	}

	// read the message from our body
	msg, err := store.MsgFromJson(r.Body)
	defer msg.Release()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// assign our connection UUID
	msg.ConnUuid = conn_uuid

	// write it out
	err = msg.WriteToOutbox()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// dispatch it
	engine.Dispatcher.Outgoing <- msg.Id

	// output it
	js, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func readMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	connUuid := ps.ByName("conn_uuid")

	msgId, err := strconv.ParseUint(ps.ByName("msg_uuid"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// load our msg and status
	msg, err := store.MsgFromId(connUuid, msgId)
	defer msg.Release()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// output it
	js, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func serveIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.ServeFile(w, r, "static/index.html")
}

var engines map[string]*engine.ConnectionEngine

func StartServer(e *map[string]*engine.ConnectionEngine) {
	engines = *e

	router := httprouter.New()
	router.GET("/", serveIndex)
	router.ServeFiles("/static/*filepath", http.Dir("static"))

	router.GET("/connection", listConnections)
	router.PUT("/connection", addConnection)
	router.DELETE("/connection/:conn_uuid", deleteConnection)
	router.GET("/connection/:conn_uuid", readConnection)
	router.PUT("/connection/:conn_uuid/send", sendMessage)
	router.GET("/connection/:conn_uuid/status/:msg_uuid", readMessage)

	log.Println("")
	log.Println(fmt.Sprintf("Starting server on http://localhost:%d", cfg.Config.Server.Port))
	log.Println("\tPUT     /connection                    - Add a connection")
	log.Println("\tGET     /connection                    - List Connections")
	log.Println("\tGET     /connection/[uuid]             - Read Connection Status")
	log.Println("\tDELETE  /connection/[uuid]             - Shut down and delete a Connection")
	log.Println("")
	log.Println("\tPUT     /connection/[uuid]/send        - Send Message")
	log.Println("\tGET     /connection/[uuid]/status/[id] - Get Message Status")
	log.Println("")

	log.Println()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Config.Server.Port), router))
}
