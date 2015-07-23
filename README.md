# GoJunebug 
This is a proof of concept of using golang for an SMPP gateway like Kannel. It doesn't do much of anything yet, it is mostly for me learning Go.

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
2015/07/19 16:35:25 	GET  /connection/[uuid]/status/[id]   - Get Message Status
```

### Sender Types
Currently there are two types of senders ```echo``` which after a configurable pause, will send the message back, and ```twitter``` that will send and receive Twitter DM's.

#### Echo Config

```pause``` - integer as string, representing how many seconds to pause before sending back an echo. Can be zero for no delay.

#### Twitter Config

```username``` - string, the username of the user sending and receiving DMs
```access_token``` - string, the access token for the user sending and receiving DMs
```access_token_secret``` - string, the access token secret for the user sending and receiving DMs

### Receiver Types
Currently there is only one type of receiver ```http```, which will POST the incoming msg to the URL provided on incoming messages.

#### HTTP Config

```url``` - string, the URL to POST to with new messages

## Endpoints
All interactions with Junebug are through HTTP endpoints

### Creating Connection
```json
PUT /connection
{
  "senders": {
    "type": "echo",
    "count": 5,
    "config": {
      "pause": "1"
    }
  },
  "receivers": {
    "receiver_type": "http",
    "count": 5,
    "config": {
      "url": "http://myhost.com/receive"
    }
  }
}
```
You will receive a response containing the connection created, and it's UUID:
```json
{
  "uuid": "3958bba4-8eae-43b8-b30c-534db207b279",
  "Senders": {
    "type": "echo",
    "count": 5,
    "config": {
      "pause": "1"
    }
  },
  "Receivers": {
    "type": "http",
    "count": 5,
    "config": {
      "url": "http://myhost.com/receive"
    }
  }
}
```

### Listing Connections
```json
GET /connection
```
You will receive a list of the active connections:
```json
{
  "connections": [
    {
      "uuid": "3958bba4-8eae-43b8-b30c-534db207b279",
      "senders": {
        "type": "echo",
        "count": 5,
        "config": {
          "pause": "1"
        }
      },
      "receivers": {
        "type": "http",
        "count": 5,
        "config": {
          "url": "http://myhost.com/receive"
        }
      }
    }
  ]
}
```

### Getting the status of a connection
```
GET /connection/[connection_uuid]
```
You will receive the connection configuration as well as it's status of queued incoming and outgoing messages:
```json
{
  "connection: {
  "uuid": "3958bba4-8eae-43b8-b30c-534db207b279",
  "Senders": {
    "type": "echo",
    "count": 5,
    "config": {
      "pause": "1"
    }
  },
  "Receivers": {
    "type": "http",
    "count": 5,
    "config": {
      "url": "http://myhost.com/receive"
    }
   }
  },
  "status": {
    "outgoing_queued": 0,
    "incoming_queued": 0,
    "handled_results": 1050,
    "sent_results": 1050
  }
}
```

### Deleting a Connection
```json
DELETE /connection/[connection_uuid]
```
You will receive a response containing the connection status when it was closed. Note that this call blocks until all workers have stopped sending, so may take a bit of time.
```json
{
  "uuid": "3958bba4-8eae-43b8-b30c-534db207b279",
  "Senders": {
    "type": "echo",
    "count": 5,
    "config": {
      "pause": "1"
    }
  },
  "Receivers": {
    "type": "http",
    "count": 5,
    "config": {
      "url": "http://myhost.com/receive"
    }
  }
}
```

### Sending a message
```json
PUT /connection/[connection_uuid]/send
{
  "text": "Hello World",
  "address": "+250788383383",
  "priority": "H"
}
```
You can pick either H (high) or L (low) as a priority. All high priority messages will be sent before any low priority messages.

You will receive the message created and its UUID:
```json
{
  "id": 2047,
  "conn_uuid": "54b7647b-924d-4ba0-b248-1145b96aefc9",
  "address": "+250788383383",
  "text": "Hello World"
  "priority": "H",
  "status": "Q",
  "log": "",
  "created": "2015-07-21T12:53:02.670865736-04:00",
  "finished": "0001-01-01T00:00:00Z"
}
```

### Checking the status of a message
```json
GET /connection/[connection_uuid]/status/[id]
```
You will receive the message content, its current status, when we finished
handling it as well as any log set by the sender or receiver.
```json
{
  "id": 2047,
  "conn_uuid": "a0b46933-aab8-4907-bee6-db6db8057bec",
  "address": "+250788383383",
  "text": "Hello world",
  "priority": "H",
  "status": "S",
  "log": "XXXX YYYY ZZZZ AAAA This is a log.\\\nXXXX YYYY ZZZZ BBBB It is fake.",
  "created": "2015-07-21T13:08:36.214434765-04:00",
  "finished": "2015-07-21T13:11:08.88047792-04:00"
}
```
