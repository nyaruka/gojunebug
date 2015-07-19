package conn

import (
  "github.com/nyaruka/junebug/disp"
  "github.com/nyaruka/junebug/msg"
  "log"
  "encoding/json"
  "net/http"
  "bytes"
)

// Http Receiver is a basic receiver that forwards the incoming message to an endpoint
type HttpReceiver struct {
  id int
  connectionConfig ConnectionConfig
  readyReceivers chan disp.MsgReceiver
  pendingJob chan disp.MsgJob
}

func (s HttpReceiver) Receive(msg disp.MsgJob){
  s.pendingJob<-msg
}

// Starts our receiver, this starts a goroutine that blocks on msgs to forward
func (r HttpReceiver) Start(){
  go func(){
    for {
      // mark ourselves as ready for work
      r.readyReceivers<-r
      log.Printf("[%s][%d] Receiver Ready", r.connectionConfig.Uuid, r.id)

      // wait for a job to come in
      job := <-r.pendingJob

      log.Printf("[%s][%d] Assigned msg (%s)", r.connectionConfig.Uuid, r.id, job.Uuid)

      // load our msg
      message, err := msg.MsgFromInbox(r.connectionConfig.Uuid, job.Uuid)
      if err != nil {
        log.Printf("[%s][%d] Error receiving msg (%s)", r.connectionConfig.Uuid, r.id, job.Uuid)
      } else {
        js, err := json.Marshal(message)
        if err != nil {
          log.Printf("[%s][%d] Error receiving msg (%s)", r.connectionConfig.Uuid, r.id, job.Uuid)
          continue
        }

        // we post our Msg body to our receiver URL
        log.Printf("[%s][%d] Post msg (%s) to %s",
                   r.connectionConfig.Uuid, r.id, job.Uuid, r.connectionConfig.ReceiverUrl)
        req, err := http.NewRequest("POST", r.connectionConfig.ReceiverUrl, bytes.NewBuffer(js))
        req.Header.Set("Content-Type", "application/json")

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
          log.Printf("[%s][%d] Error delivering incoming msg (%s)", r.connectionConfig.Uuid, r.id, job.Uuid)
          log.Printf(err.Error())
        } else {
          if resp.Status != "200" || resp.Status != "201" {
            log.Printf("[%s][%d] Error delivering incoming msg (%s) got status (%s)",
                       r.connectionConfig.Uuid, r.id, job.Uuid, resp.Status)
          }
          resp.Body.Close()
        }
      }

      // mark the message as sent
      err = message.MarkHandled()
      log.Printf("[%s][%d] Handled msg (%s)", r.connectionConfig.Uuid, r.id, job.Uuid)
      if err != nil {
        log.Println("Error marking msg handled")
      }
    }
  }()
}

func CreateHttpReceiver(id int, config ConnectionConfig, readyReceivers chan disp.MsgReceiver) HttpReceiver {
  return HttpReceiver{ id: id,
                       connectionConfig: config,
                       readyReceivers: readyReceivers,
                       pendingJob: make(chan disp.MsgJob) }
}
