package conn

import (
  "github.com/nyaruka/junebug/disp"
  "github.com/nyaruka/junebug/msg"
  "log"
  "time"
)

// EchoSender is a dummy sender that takes 5 seconds to send anything, then returns an
// echo of the sent message back through our connection.
//
// It is an implementation of MsgSender

type EchoSender struct {
  id int
  connectionConfig ConnectionConfig
  readySenders chan disp.MsgSender
  pendingJob chan disp.MsgJob
  incoming chan disp.MsgJob // for receiving out our echos
}

func (s EchoSender) Send(msg disp.MsgJob){
  s.pendingJob<-msg
}

// Starts our sender, this starts a goroutine that blocks on receiving a message to send
func (s EchoSender) Start(){
  go func(){
    for {
      // mark ourselves as ready for work
      s.readySenders<-s
      log.Printf("[%s][%d] Sender Ready", s.connectionConfig.Uuid, s.id)

      // wait for a job to come in
      job := <-s.pendingJob

      log.Printf("[%s][%d] Assigned msg (%s)", s.connectionConfig.Uuid, s.id, job.Uuid)

      // load our msg
      message, err := msg.MsgFromOutbox(s.connectionConfig.Uuid, job.Uuid)
      if err != nil {
        log.Printf("[%s][%d] Error sending msg (%s)", s.connectionConfig.Uuid, s.id, job.Uuid)
      } else {
        // sleep a bit to slow things down
        time.Sleep(time.Second * 5)
      }

      // mark the message as sent
      err = message.MarkSent()
      if err != nil {
        log.Printf("[%s][%d] Error marking msg sent (%s)", s.connectionConfig.Uuid, s.id, job.Uuid)
      } else {
        log.Printf("[%s][%d] Sent msg (%s)", s.connectionConfig.Uuid, s.id, job.Uuid)
      }

      // create a new incoming msg
      incoming := msg.MsgFromText(s.connectionConfig.Uuid, message.Address, "echo: " + message.Text)
      _, err = incoming.WriteToInbox()
      if err != nil {
        log.Printf("[%s][%d] Error add incoming msg (%s)", s.connectionConfig.Uuid, s.id, incoming.Uuid)
      }

      s.incoming <- disp.MsgJob{incoming.Uuid}
    }
  }()
}

func CreateEchoSender(id int, config ConnectionConfig, readySenders chan disp.MsgSender, incoming chan disp.MsgJob) EchoSender {
  return EchoSender{ id: id,
                     connectionConfig: config,
                     readySenders: readySenders,
                     incoming: incoming,
                     pendingJob: make(chan disp.MsgJob) }
}
