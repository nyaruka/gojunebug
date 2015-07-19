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
2015/07/19 16:35:25 	POST /connection                      - Add a connection
2015/07/19 16:35:25 	GET  /connection                      - List Connections
2015/07/19 16:35:25 	GET  /connection/[uuid]               - Read Connection Status
2015/07/19 16:35:25 	POST /connection/[uuid]/send          - Send Message
2015/07/19 16:35:25 	GET  /connection/[uuid]/status/[uuid] - Get Message Status
```

## Endpoints
All interactions with Junebug are through HTTP endpoints

### Creating Connection
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
You will receive a response containing the connection created, and it's UUID:
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

### Listing Connections
```
GET /connection
```
You will receive a list of the active connections:
```
{
  "connections": [
    {
      "uuid": "54b7647b-924d-4ba0-b248-1145b96aefc9",
      "sender_type": "echo",
      "num_senders": 5,
      "receiver_type": "http",
      "num_receivers": 5,
      "receiver_url": "http://myhost.com/receive"
    }
  ]
}
```

### Getting the status of a connection
```
GET /connection/[connection_uuid]
```
You will receive the connection configuration as well as it's status of queued incoming and outgoing messages:
```
{
  "connection": {
    "uuid": "54b7647b-924d-4ba0-b248-1145b96aefc9",
    "sender_type": "echo",
    "num_senders": 5,
    "receiver_type": "http",
    "num_receivers": 5,
    "receiver_url": "http://myhost.com/receive"
  },
  "status": {
    "outgoing_queued": 0,
    "incoming_queued": 0
  }
}
```

### Sending a message
```
POST /connection/[connection_uuid]/send
{
  "text": "Hello World",
  "address": "+250788383383"
}
```
You will receive the message created and its UUID:
```
{
  "uuid": "2ac20704-bd15-4299-ad6c-0d1892ae54e8",
  "conn_uuid": "54b7647b-924d-4ba0-b248-1145b96aefc9",
  "address": "+250788383383",
  "text": "Hello World"
}
```

### Checking the status of a message
```
GET /connection/[connection_uuid]/status/[msg_uuid]
```
You will receive the message content and its current status
```
{
  "message": {
    "uuid": "2ac20704-bd15-4299-ad6c-0d1892ae54e8",
    "conn_uuid": "54b7647b-924d-4ba0-b248-1145b96aefc9",
    "address": "+250788383383",
    "text": "Hello World"
  },
  "status": {
    "status": "sent"
  }
}
```
