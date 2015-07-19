# GoJunebug
This is a proof of concept of using golang for an SMPP gateway like Kannel. It is just a proof of concept and 
doesn't actually do all that much yet.

## Running
First update junebug.conf to have the appropriate settings. Specifically make sure all the revelant directories exit.

```bash
% export GOPATH=`pwd`
% go install github.com/nyaruka/junebug
% ./bin/junebug --settings=junebug.conf
2015/07/19 16:35:25 Starting server on http://localhost:8000
2015/07/19 16:35:25 	POST /connection                    - Add a connection
2015/07/19 16:35:25 	GET  /connection                    - List Connections
2015/07/19 16:35:25 	GET  /connection/[uuid]             - Read Connection Status
2015/07/19 16:35:25 	POST /connection/[uuid]/send        - Send Message
2015/07/19 16:35:25 	GET  /connection/[uuid]/send/[uuid] - Get Message Status
```

## Endpoints
All interactions with Junebug are through HTTP endpoints

### Creation Connection
```
POST /connection
{
  "receiver_type": "http",
  "num_receivers": 5,
  "receiver_url": "http://myhost.com/receive",
  "sender_type": "echo",
  "num_senders": 5
}
```
You will receive a response containing the configuration created, and it's UUID:
```
{
  "uuid": "54b7647b-924d-4ba0-b248-1145b96aefc9",
  "sender_type": "echo",
  "num_senders": 5,
  "receiver_type": "http",
  "num_receivers": 5,
  "receiver_url": "http://myhost.com/receive"
}
```
