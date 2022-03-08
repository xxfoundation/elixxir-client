# xx network Client

[![pipeline status](https://gitlab.com/elixxir/client/badges/master/pipeline.svg)](https://gitlab.com/elixxir/client/commits/master)
[![coverage report](https://gitlab.com/elixxir/client/badges/master/coverage.svg)](https://gitlab.com/elixxir/client/commits/master)

The xx network client interfaces with the cMix system, enabling access
to all of the xx network messaging features, including end-to-end encryption and metadata protection. 

The client is a library and related command line tool 
that facilitate making full-featured xx clients for all platforms. The
command line tool can be built for any platform supported by
golang. The libraries are built for iOS and Android using
[gomobile](https://godoc.org/golang.org/x/mobile/cmd/gomobile).

This repository contains everything necessary to implement all of the
xx network messaging features. It also contains features to extend the base 
messaging protocols.

For library writers, the client requires a writable folder to store
data, functions for receiving and approving requests for creating
secure end-to-end messaging channels, for discovering users, and for
receiving different types of messages. Details for implementing these
features are in the [Library Overview section](#library-overview) below.

The client is open source software released under the simplified BSD License.

## Command Line Usage

The command line tool is intended for testing xx network functionality and not
for regular user use. 

These instructions assume that you have [Go 1.17.X installed](https://go.dev/doc/install), and GCC installed for Cgo (such as `build-essential` on Debian or Ubuntu).

Compilation steps:

```
git clone https://gitlab.com/elixxir/client.git client
cd client
go mod vendor -v
go mod tidy
go test ./...
# Linux 64 bit binary
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o client.linux64 main.go
# Windows 64 bit binary
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o client.win64 main.go
# Windows 32 big binary
GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -ldflags '-w -s' -o release/client.win32 main.go
# Mac OSX 64 bit binary (intel)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o release/client.darwin64 main.go
```

#### Fetching an NDF

All actions performed with the client require a current [NDF](https://xxdk-dev.xx.network/technical-glossary#network-definition-file-ndf). The NDF is downloadable from the command line or via an access point from the Client API.

To fetch the NDF via the command  line, use the `getndf` command. `getndf` enables command line users to poll the NDF from a network gateway without any pre-established client connection:
```
// Fetch NDF (example usage for Gateways, assumes you are running a gateway locally)
$ go run main.go getndf --gwhost localhost:8440 --cert certfile.pem | jq . >ndf.json
```

You can also download an NDF directly for different environments by using the `--env` flag:

```go
$ go run main.go getndf --env mainnet | jq . >ndf.json
// Or, run via the binary (assuming 64-bit Windows): 
$ client getndf --env mainnet | jq . >ndf.json
```

Sample output of `ndf.json`:
```
{
  "Timestamp": "2021-01-29T01:19:49.227246827Z",
  "Gateways": [
    {
      "Id": "BRM+Iotl6ujIGhjRddZMBdauapS7Z6jL0FJGq7IkUdYB",
      "Address": ":8440",
      "Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJp\ncDAeFw0xOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVT\nMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNV\nBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIw\nDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDh\nDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfs\nWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSE\ntJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uA\nm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9\nbJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEA\nAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEA\nneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIf\nU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2\nqvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4\ncyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1R\ntgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E5\n6m52PyzMNV+2N21IPppKwA==\n-----END CERTIFICATE-----\n"
    },
    {
      "Id": "JCBd9mAQb2BW8hc8H9avy1ubcjUAa7MHrPp0dBU/VqQB",
	  .....
```

#### Sending safe messages between 2 users

To send messages with end to end encryption, you must first establish a connection
or [authenticated channel](https://xxdk-dev.xx.network/technical-glossary#authenticated-channel) with the other user:

```
# Get user contact jsons for each client
$ client --password user1-password --ndf ndf.json -l client1.log -s user1session --writeContact user1-contact.json --unsafe -m "Hi to me, without E2E Encryption"
$ client --password user2-password --ndf ndf.json -l client2.log -s user2session --writeContact user2-contact.json --unsafe -m "Hi to me, without E2E Encryption"

# Request authenticated channel from other client. Note that the receiving client
# is expected to confirm request before any specified timeout (default 120s)
$ client --password password --ndf ndf.json -l client.log -s session-directory --destfile user2-contact.json --waitTimeout 360 --unsafe-channel-creation --send-auth-request
WARNING: unsafe channel creation enabled
Adding authenticated channel for: Qm40C5hRUm7uhp5aATVWhSL6Mt+Z4JVBQrsEDvMORh4D
Message received:
Sending to Qm40C5hRUm7uhp5aATVWhSL6Mt+Z4JVBQrsEDvMORh4D:
Received 1

# Alternatively, to accept an authenticated channel request implicitly
# (should be within the timeout window of requesting client or the request will need to be resent):
$ client --password "password" --ndf ndf.json -l client.log -s session-directory --destfile user2-contact.json" --unsafe-channel-creation --waitTimeout 200
Authentication channel request from: o+QpswTmnsuZve/QRz0j0RYNWqjgx4R5pACfO00Pe0cD
Sending to o+QpswTmnsuZve/QRz0j0RYNWqjgx4R5pACfO00Pe0cD:
Message received:
Received 1

# Send E2E Messages
$ client --password user1-password --ndf ndf.json -l client1.log -s user1session --destfile user2-contact.json -m "Hi User 2, from User 1 with E2E Encryption"
Sending to Qm40C5hRUm7uhp5aATVWhSL6Mt+Z4JVBQrsEDvMORh4D: Hi User 2, from User 1 with E2E Encryption
Timed out!
Received 0

$ client --password user2-password --ndf ndf.json -l client1.log -s user2session --destfile user1-contact.json -m "Hi User 1, from User 2 with E2E Encryption"
Sending to o+QpswTmnsuZve/QRz0j0RYNWqjgx4R5pACfO00Pe0cD: Hi User 1, from User 2 with E2E Encryption
Timed out!
Received 0
```

* `--password`: The password used to encrypt and load the session.
* `--ndf`: The network definition file.
* `-l`: The file to write logs (user messages are still printed to stdout).
* `-s`: The storage directory for client session data.
* `--writeContact`: Output the user's contact information to this file.
* `--destfile` is used to specify the recipient. You can also use
  `--destid b64:...` using the user's base64 id which is printed in the logs.
* `--unsafe`: Send message without encryption (necessary whenever you have not
  already established an e2e channel).
* `--unsafe-channel-creation` Auto-create and auto-accept channel requests.
* `-m`: The message to send.

Note that the client defaults to sending to itself when a destination is not supplied.
This is why we've used the `--unsafe` flag when creating the user contact jsons.
However when sending between users, it is dropped in exchange for `--unsafe-channel-creation`.

To be considered "safe" the user should be prompted. You can do this
on the command line by explicitly accepting the channel creation
when sending and/or explicitly accepting a request with
`--accept-channel`:

```
$ client --password user-password --ndf ndf.json -l client.log -s session-directory --destfile user-contact.json --accept-channel
Authentication channel request from: yYAztmoCoAH2VIr00zPxnj/ZRvdiDdURjdDWys0KYI4D
Sending to yYAztmoCoAH2VIr00zPxnj/ZRvdiDdURjdDWys0KYI4D:
Message received:
Received 1
```

Full usage of client can be found with `client --help`:

```
$ ./client --help
Runs a client for cMix anonymous communication platform

Usage:
  client [flags]
  client [command]

Available Commands:
  fileTransfer Send and receive file for cMix client
  generate     Generates version and dependency information for the Elixxir binary
  getndf       Download the network definition file from the network and print it.
  group        Group commands for cMix client
  help         Help about any command
  init         Initialize a user ID but do not connect to the network
  single       Send and respond to single-use messages.
  ud           Register for and search users using the xx network user discovery service.
  version      Print the version and dependency information for the Elixxir binary

Flags:
      --accept-channel            Accept the channel request for the corresponding recipient ID
      --auth-timeout uint         The number of seconds to wait for an authentication channelto confirm (default 120)
      --delete-all-requests       Delete the all contact requests, both sent and received.
      --backupIn string           Path to load backup client from
      --backupOut string          Path to output backup client.
      --backupPass string         Passphrase to encrypt/decrypt backup
      --delete-channel            Delete the channel information for the corresponding recipient ID
      --delete-receive-requests   Delete the all received contact requests.
      --delete-sent-requests      Delete the all sent contact requests.
      --destfile string           Read this contact file for the destination id
  -d, --destid string             ID to send message to (if below 40, will be precanned. Use '0x' or 'b64:' for hex and base64 representations) (default "0")
      --e2eMaxKeys uint           Max keys used before blocking until a rekey completes (default 800)
      --e2eMinKeys uint           Minimum number of keys used before requesting rekey (default 500)
      --e2eNumReKeys uint         Number of rekeys reserved for rekey operations (default 16)
      --e2eRekeyThreshold float64 Number between 0 an 1. Percent of keys used before a rekey is started
      --forceHistoricalRounds     Force all rounds to be sent to historical round retrieval
      --forceMessagePickupRetry   Enable a mechanism which forces a 50% chance of no message pickup, instead triggering the message pickup retry mechanism
  -h, --help                      help for client
  -l, --log string                Path to the log output path (- is stdout) (default "-")
  -v, --logLevel uint             Verbose mode for debugging
  -m, --message string            Message to send
  -n, --ndf string                Path to the network definition JSON file (default "ndf.json")
  -p, --password string           Password to the session file
      --profile-cpu string        Enable cpu profiling to this file
      --protoUserOut string       Path to which a normally constructed client will write proto user JSON file
      --protoUserPath string      Path to proto user JSON file containing cryptographic primitives the client will load
      --receiveCount uint         How many messages we should wait for before quitting (default 1)
      --regcode string            Identity code (optional)
      --send-auth-request         Send an auth request to the specified destination and waitfor confirmation
      --sendCount uint            The number of times to send the message (default 1)
      --sendDelay uint            The delay between sending the messages in ms (default 500)
      --sendid uint               Use precanned user id (must be between 1 and 40, inclusive)
  -s, --session string            Sets the initial storage directory for client session data
      --slowPolling               Enables polling for unfiltered network updates with RSA signatures
      --unsafe                    Send raw, unsafe messages without e2e encryption.
      --unsafe-channel-creation   Turns off the user identity authenticated channel check, automatically approving authenticated channels
      --verboseRoundTracking      Verbose round tracking, keeps track and prints all rounds the client was aware of while running. Defaults to false if not set.
      --verify-sends              Ensure successful message sending by checking for round completion
      --waitTimeout uint          The number of seconds to wait for messages to arrive (default 15)
  -w, --writeContact string       Write contact information, if any, to this file,  defaults to stdout (default "-")

Use "client [command] --help" for more information about a command.
```

**Note:** The client cannot be used on the betanet with precanned user ids.

## Library Overview

The xx client is designed to be used as a go library (and by extension a 
c library). 
 
Support is also present for go mobile to build Android and iOS libraries. We
bind all exported symbols from the bindings package for use on mobile
platforms.

### Implementation Notes

Clients need to perform the same actions *in the same order* as shown in
`cmd/root.go`. Specifically, certain handlers need to be registered and
set up before starting network threads. Additionally, you cannot perform certain actions until the network connection reaches the "healthy" state.

See [main.go](https://git.xx.network/elixxir/xxdk-examples/-/blob/sample-messaging-app/sample-messaging-app/main.go) for relevant code listings on when and how to perform these actions.
The [Getting Started](https://xxdk-dev.xx.network/getting-started) guide provides further detail.

You can also visit the [API Quick Reference](https://xxdk-dev.xx.network/quick-reference)
for information on the types and functions exposed by the Client API.

The main entry point for developing with the client is `api/client` (or
`bindings/client`). We recommend using go doc to explore:

```
go doc -all ./api
go doc -all ./interfaces
```

Looking at the API will, for example, show you there is a RoundEvents callback
registration function, which lets your client see round events:

```
func (c *Client) GetRoundEvents() interfaces.RoundEvents
    RegisterRoundEventsCb registers a callback for round events.
```

and then inside interfaces:

```
type RoundEvents interface {
        // designates a callback to call on the specified event
        // rid is the id of the round the event occurs on
        // callback is the callback the event is triggered on
        // timeout is the amount of time before an error event is returned
        // valid states are the states which the event should trigger on
        AddRoundEvent(rid id.Round, callback ds.RoundEventCallback,
                timeout time.Duration, validStates ...states.Round) *ds.EventCallback

        // designates a go channel to signal the specified event
        // rid is the id of the round the event occurs on
        // eventChan is the channel the event is triggered on
        // timeout is the amount of time before an error event is returned
        // valid states are the states which the event should trigger on
        AddRoundEventChan(rid id.Round, eventChan chan ds.EventReturn,
                timeout time.Duration, validStates ...states.Round) *ds.EventCallback

        //Allows the un-registration of a round event before it triggers
        Remove(rid id.Round, e *ds.EventCallback)
}
```

Which, when investigated, yields the following prototype:

```
// Callbacks must use this function signature
type RoundEventCallback func(ri *pb.RoundInfo, timedOut bool)
```

showing that you can receive a full RoundInfo object for any round event
received by the client library on the network.

### Building the Library for iOS and Android

To set up Gomobile for Android, install the NDK and pass the -ndk flag
to ` $ gomobile init`. Other repositories that use Gomobile for
binding should include a shell script that creates the bindings. For
iOS, gomobile must be run on an OS X machine with Xcode installed.

Important reference info:
1. [Setting up Gomobile and subcommands](https://godoc.org/golang.org/x/mobile/cmd/gomobile)
2. [Reference cycles, type restrictions](https://godoc.org/golang.org/x/mobile/cmd/gobind)

To clone and build:

```
# Go mobile install
go get -u golang.org/x/mobile/cmd/gomobile
go get -u golang.org/x/mobile/bind
gomobile init... # Note this line will be different depending on sdk/target!
# Get and test code
git clone https://gitlab.com/elixxir/client.git client
cd client
go mod vendor -v
go mod tidy
go test ./...
# Android
gomobile bind -target android -androidapi 21 gitlab.com/elixxir/client/bindings
# iOS
gomobile bind -target ios gitlab.com/elixxir/client/bindings
zip -r iOS.zip Bindings.framework
```

You can verify that all symbols got bound by unzipping
`bindings-sources.jar` and inspecting the resulting source files.

Every time you make a change to the client or bindings, you must
rebuild the client bindings into a .aar or iOS.zip to propagate those
changes to the app. There's a script that runs gomobile for you in the
`bindings-integration` repository.

## Roadmap

See the larger network documentation for more, but there are 2 specific
parts of the roadmap that are intended for the client:

* Ephemeral IDs - Sending messages to users with temporal/ephemeral recipient
  user identities.
* User Discovery - A bot that will allow the user to look for others on the
  network.
* Notifications - An optional notifications system which uses firebase
* Efficiency improvements - mechanisms for message pickup and network tracking 
* will evolve to allow tradeoffs and options for use

We also are always looking at how to simplify and improve the library interface.
