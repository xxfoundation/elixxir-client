# xx network Client

[Repository](https://git.xx.network/elixxir/client)
| [Go Doc](https://pkg.go.dev/gitlab.com/elixxir/client/xxdk)
| [Examples](https://git.xx.network/elixxir/xxdk-examples/-/tree/master)

The client is a library and related command-line tool that facilitates making full-featured xx clients for all
platforms. It interfaces with the cMix system, enabling access to all xx network messaging features, including
end-to-end encryption and metadata protection.

This repository contains everything necessary to implement the xx network messaging features. In addition, it also
contains features to extend the base messaging protocols.

The command-line tool accompanying the client library can be built for any platform supported by Go. The libraries are
built for iOS and Android using [gomobile](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile). The libraries are built
for web assembly using the repository [xxdk-wasm](https://git.xx.network/elixxir/xxdk-wasm).

For library writers, the client requires a writable folder to store data, functions for receiving and approving requests
for creating secure end-to-end messaging channels, discovering users, and receiving different types of messages.

The client is open-source software released under the simplified BSD License.

## Command Line Usage

The command-line tool is intended for testing xx network functionality and not for regular user use.

These instructions assume that you have [Go 1.17.X](https://go.dev/doc/install) installed and GCC installed for
[cgo](https://pkg.go.dev/cmd/cgo) (such as `build-essential` on Debian or Ubuntu).

Compilation steps:

```shell
$ git clone https://gitlab.com/elixxir/client.git client
$ cd client
$ go mod vendor
$ go mod tidy

# Linux 64-bit binary
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o client.linux64 main.go

# Windows 64-bit binary
$ GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o client.win64 main.go

# Windows 32-bit binary
$ GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -ldflags '-w -s' -o client.win32 main.go

# Mac OSX 64-bit binary (Intel)
$ GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o client.darwin64 main.go
```

### Fetching an NDF

All actions performed with the client require a current
[network definition file (NDF)](https://xxdk-dev.xx.network/technical-glossary/#network-definition-file-ndf). The NDF is
downloadable from the command line or
[via an access point](https://xxdk-dev.xx.network/quick-reference#func-downloadandverifysignedndfwithurl) in the Client
API.

Use the `getndf` command to fetch the NDF via the command line. `getndf` enables command-line users to poll the NDF from
a network gateway without any pre-established client connection.

First, you'll want to download an SSL certificate:

```shell
# Assumes you are running a gateway locally
$ openssl s_client -showcerts -connect localhost:8440 < /dev/null 2>&1 | openssl x509 -outform PEM > certfile.pem
```

Now you can fetch the NDF.

```shell
# Example usage for gateways, assumes you are running a gateway locally
$ ./client getndf --gwhost localhost:8440 --cert certfile.pem | jq . > ndf.json
```

You can also download an NDF directly for different environments by using the `--env` flag.

```shell
$ ./client getndf --env mainnet | jq . > ndf.json
```

Sample content of `ndf.json`:

```json
{
	"Timestamp": "2021-01-29T01:19:49.227246827Z",
	"Gateways": [
		{
			"Id": "BRM+IoTl6ujIGhjRddZMBdaUapS7Z6jL0FJGq7IkUdYB",
			"Address": ":8440",
			"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDbDCCAlSgAwIBAgIJA8UNtZneIYE2MA0GCSqGSIb3DQE3BQU8MGgxCzAJBgNV\nBaYTAlVTmRMwEQYDvQQiDApDYWxpZm9ybmLhMRIwEAYfVQqHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBeGA1UEAwwKKi5jbWl4LnJp\ncDAeFw0xOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVT\nMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNV\nBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIw\nDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDh\nDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfs\nWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSE\ntJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uA\nm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9\nbJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEA\nAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEA\nneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIf\nU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2\nqvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4\ncyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1R\ntgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E5\n6m52PyzMNV+2N21IPppKwA==\n-----END CERTIFICATE-----\n"
		}
	]
}
```

Use the `--help` command with `getndf` to view all options.

```shell
$ ./client getndf --help
```

### Sending Safe Messages Between Two (2) Users

> 💡 **Note:** For information on receiving messages and troubleshooting authenticated channel requests, see
> [Receiving Messages](#receiving-messages)
> and [Confirming authenticated channel requests](#confirming-authenticated-channel-requests).

To send messages with end-to-end encryption, you must first establish a connection
or [authenticated channel](https://xxdk-dev.xx.network/technical-glossary#authenticated-channel) with the other user.
See below for example commands for sending or confirming authenticated channel requests, as well as for sending E2E
messages.

```shell
# Get user contact files for each client
$ ./client --password user1-password --ndf ndf.json -l client1.log -s user1session --writeContact user1-contact.json --unsafe -m "Hi to me, without E2E Encryption"
$ ./client --password user2-password --ndf ndf.json -l client2.log -s user2session --writeContact user2-contact.json --unsafe -m "Hi to me, without E2E Encryption"

# Request authenticated channel from another client. Note that the receiving client
# is expected to confirm the request before any specified timeout (default 120s)
$ ./client --password password --ndf ndf.json -l client.log -s session-directory --destfile user2-contact.json --waitTimeout 360 --unsafe-channel-creation --send-auth-request
WARNING: unsafe channel creation enabled
Adding authenticated channel for: Qm40C5hRUm7uhp5aATVWhSL6Mt+Z4JVBQrsEDvMORh4D
Message received:
Sending to Qm40C5hRUm7uhp5aATVWhSL6Mt+Z4JVBQrsEDvMORh4D:
Received 1

# Accept/Confirm an authenticated channel request implicitly
# (should be within the timeout window of requesting client, or the request will need to be re-sent):
$ ./client --password "password" --ndf ndf.json -l client.log -s session-directory --destfile user2-contact.json --unsafe-channel-creation --waitTimeout 200
Authentication channel request from: o+QpswTmnsuZve/QRz0j0RYNWqjgx4R5pACfO00Pe0cD
Sending to o+QpswTmnsuZve/QRz0j0RYNWqjgx4R5pACfO00Pe0cD:
Message received:
Received 1

# Send E2E Messages
$ ./client --password user1-password --ndf ndf.json -l client1.log -s user1session --destfile user2-contact.json -m "Hi User 2, from User 1 with E2E Encryption"
Sending to Qm40C5hRUm7uhp5aATVWhSL6Mt+Z4JVBQrsEDvMORh4D: Hi User 2, from User 1 with E2E Encryption
Timed out!
Received 0

$ ./client --password user2-password --ndf ndf.json -l client1.log -s user2session --destfile user1-contact.json -m "Hi User 1, from User 2 with E2E Encryption"
Sending to o+QpswTmnsuZve/QRz0j0RYNWqjgx4R5pACfO00Pe0cD: Hi User 1, from User 2 with E2E Encryption
Timed out!
Received 0
```

* `--password`: The password used to encrypt and load the session.
* `--ndf`: The network definition file.
* `-l`: The file to write logs (user messages are still printed to stdout).
* `-s`: The storage directory for client session data.
* `--writeContact`: Output the user's contact information to this file.
* `--destfile` is used to specify the recipient. You can also use `--destid b64:...` using the user's base64 ID, which
  is printed in the logs.
* `--unsafe`: Send message without encryption (necessary whenever you have not already established an e2e channel).
* `--unsafe-channel-creation` Auto-create and auto-accept channel requests.
* `-m`: The message to send.

Note that the client defaults to sending to itself when a destination is not supplied.
This is why we've used the `--unsafe` flag when creating the user contact files.
However, when sending between users, the flag is dropped in exchange for `--unsafe-channel-creation`.

For the authenticated channel creation to be considered "safe", the user should be prompted. You can do this by
explicitly accepting the channel creation when sending a request with `--send-auth-request` (while excluding the
`--unsafe-channel-creation` flag) or explicitly accepting a request with `--accept-channel`:

```shell
$ ./client --password user-password --ndf ndf.json -l client.log -s session-directory --destfile user-contact.json --accept-channel
Authentication channel request from: yYAztmoCoAH2VIr00zPxnj/ZRvdiDdURjdDWys0KYI4D
Sending to yYAztmoCoAH2VIr00zPxnj/ZRvdiDdURjdDWys0KYI4D:
Message received:
Received 1
```

### Receiving Messages

There is no explicit command for receiving messages. Instead, the client will attempt to fetch pending messages on each
run.

You can use the `--receiveCount` flag to limit the number of messages the client waits for before a timeout occurs:

```shell
$ ./client --password <password> --ndf <NDF JSON file> -l client.log -s <session directory> --destfile <contact JSON file> --receiveCount <integer count>
```

### Sending Authenticated Channel Requests

See [Sending Safe Messages Between Two (2) Users](#sending-safe-messages-between-two-2-users)

### Confirming Authenticated Channel Requests

Setting up an authenticated channel between clients is a back-and-forth process that happens in sequence. One client
sends a request and waits for the other to accept it.

See the previous section, [Sending safe messages between 2 users](#sending-safe-messages-between-two-2-users), for
example, commands showing how to set up an end-to-end connection between clients before sending messages.

As with received messages, there is no command for checking for authenticated channel requests; you'll be notified of
any pending requests whenever the client is run.

```shell
$ ./client.win64 --password password --ndf ndf.json -l client.log -s session-directory --destfile user-contact8.json --waitTimeout 120 -m "Hi User 7, from User 8 with E2E Encryption"
Authentication channel request from: 8zAWY69UUK/FkMBGY3ViR5MMfcp1GoKn6Y3c/64NYNYD
Sending to yYAztmoCoAH2VIr00zPxnj/ZRvdiDdURjdDWys0KYI4D: Hi User 7, from User 8 with E2E Encryption
Timed out!
Received 0

```

### Troubleshooting

**`panic: Could not confirm authentication channel for ...`**

Suppose the receiving client does not confirm the authentication channel before the requesting client reaches a
timeout (default 120s). In that case, the request eventually terminates in
a `panic: Could not confirm authentication channel for ...` error.

Retrying the request should fix this. If necessary, you may increase the time the client waits to confirm the channel
before timeout using the `--auth-timeout` flag (default 120s).

This error will also occur with the receiving client if it received the request but failed to confirm it before the
requesting client reached a timeout. In this case, the request must be resent while the other client reattempts to
confirm the channel.

**`panic: Received request not found`**

You may also run into the `panic: Received request not found` error when attempting to confirm an authenticated channel
request. This means your client has not received the request. If one has been sent, simply retrying should fix this.

Full usage of the client can be found with `client --help`:

```text
$ ./client --help
Runs a client for cMix anonymous communication platform

Usage:
  client [flags]
  client [command]

Available Commands:
  broadcast    Send broadcast messages
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
      --auth-timeout uint         The number of seconds to wait for an authentication channelto confirm (default 60)
      --backupIdList string       JSON file containing the backed up partner IDs
      --backupIn string           Path to load backup client from
      --backupJsonOut string      Path to output unencrypted client JSON backup.
      --backupOut string          Path to output encrypted client backup. If no path is supplied, the backup system is not started.
      --backupPass string         Passphrase to encrypt/decrypt backup
      --delete-all-requests       DeleteFingerprint the all contact requests, both sent and received.
      --delete-channel            DeleteFingerprint the channel information for the corresponding recipient ID
      --delete-receive-requests   DeleteFingerprint the all received contact requests.
      --delete-request            DeleteFingerprint the request for the specified ID given by the destfile flag's contact file.
      --delete-sent-requests      DeleteFingerprint the all sent contact requests.
      --destfile string           Read this contact file for the destination id
  -d, --destid string             ID to send message to (if below 40, will be precanned. Use '0x' or 'b64:' for hex and base64 representations) (default "0")
      --e2eMaxKeys uint           Max keys used before blocking until a rekey completes (default 2000)
      --e2eMinKeys uint           Minimum number of keys used before requesting rekey (default 1000)
      --e2eNumReKeys uint         Number of rekeys reserved for rekey operations (default 16)
      --e2eRekeyThreshold float   Number between 0 an 1. Percent of keys used before a rekey is started (default 0.05)
      --force-legacy              Force client to operate using legacy identities.
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
      --regcode string            ReceptionIdentity code (optional)
      --send-auth-request         Send an auth request to the specified destination and waitfor confirmation
      --sendCount uint            The number of times to send the message (default 1)
      --sendDelay uint            The delay between sending the messages in ms (default 500)
      --sendid uint               Use precanned user id (must be between 1 and 40, inclusive)
  -s, --session string            Sets the initial storage directory for client session data
      --slowPolling               Enables polling for unfiltered network updates with RSA signatures
      --splitSends                Force sends to go over multiple rounds if possible
      --unsafe                    Send raw, unsafe messages without e2e encryption.
      --unsafe-channel-creation   Turns off the user identity authenticated channel check, automatically approving authenticated channels
      --verboseRoundTracking      Verbose round tracking, keeps track and prints all rounds the client was aware of while running. Defaults to false if not set.
      --verify-sends              Ensure successful message sending by checking for round completion
      --waitTimeout uint          The number of seconds to wait for messages to arrive (default 15)
  -w, --writeContact string       Write contact information, if any, to this file,  defaults to stdout (default "-")
      --batchMessagePickup              Enables alternate message pickup logic which processes batches
      --batchPickupDelay int            Sets the delay (in MS) before a batch pickup request is sent, even if the batch is not full (default 50)
      --batchPickupTimeout int          Sets the timeout duration (in MS) sent to gateways that proxy batch message pickup requests (default 250)
      --maxPickupBatchSize int          Set the maximum number of requests in a batch pickup message (default 20)

Use "client [command] --help" for more information about a command.
```

> 💡 **Note:**  The client cannot be used on the xx network with pre-canned user IDs.

## Library Overview

The xx client is designed to be a Go library (and, by extension, a C library).

Support is also present for Go mobile to build Android and iOS libraries. In addition, we bind all exported symbols from
the bindings package for use on mobile platforms.

This library is also supported by WebAssembly through the [xxdk-wasm](https://git.xx.network/elixxir/xxdk-wasm)
repository. xxdk-wasm wraps the bindings package in this repository so that they can be used by Javascript when compiled
for WebAssembly.

### Implementation Notes

Clients must perform the same actions *in the same order* as shown in `cmd/root.go`. Specifically, certain handlers need
to be registered and set up before starting network threads. Additionally, you cannot perform certain actions until the
network connection reaches a "healthy" state.

Refer to Setting Up a cMix Client in the API documentation for specific on how to do this.

See the [xxdk Example repository](https://git.xx.network/elixxir/xxdk-examples/-/tree/master) for various example
implementations.
In addition, the [Getting Started](https://xxdk-dev.xx.network/getting-started) guide provides further detail.

You can also visit the [API Quick Reference](https://xxdk-dev.xx.network/quick-reference) for information on the types
and functions exposed by the Client API.

The main entry point for developing with the client is `xxdk/cmix` (or `bindings/cmix`). We recommend using the
[documentation in the Go package directory](https://pkg.go.dev/gitlab.com/elixxir/client/xxdk).

If you are developing with the client through the browser and Javascript, refer to the
[xxdk-wasm](https://git.xx.network/elixxir/xxdk-wasm) repository, which wraps the `bindings/cmix` package. You may also
want to refer to the [Go documentation](https://pkg.go.dev/gitlab.com/elixxir/xxdk-wasm/wasm).

Looking at the API will, for example, show you there is a `RoundEvents` callback registration function, which lets your
client see round events.

> ```go
> func (c *Cmix) GetRoundEvents() interfaces.RoundEvents
> ```
> RegisterRoundEventsCb registers a callback for round events.

And then, inside the `RoundEvents` interfaces:

> ```go
> type RoundEvents interface {
>   // designates a callback to call on the specified event
>   // rid is the id of the round the event occurs on
>   // callback is the callback the event is triggered on
>   // timeout is the amount of time before an error event is returned
>   // valid states are the states which the event should trigger on
>   AddRoundEvent(rid id.Round, callback ds.RoundEventCallback,
>     timeout time.Duration, validStates ...states.Round) *ds.EventCallback
>   
>   // designates a go channel to signal the specified event
>   // rid is the id of the round the event occurs on
>   // eventChan is the channel the event is triggered on
>   // timeout is the amount of time before an error event is returned
>   // valid states are the states which the event should trigger on
>   AddRoundEventChan(rid id.Round, eventChan chan ds.EventReturn,
>     timeout time.Duration, validStates ...states.Round) *ds.EventCallback
>   
>   // Allows the un-registration of a round event before it triggers
>   Remove(rid id.Round, e *ds.EventCallback)
> }
> ```

Which, when investigated, yields the following prototype.

> ```go
> // Callbacks must use this function signature
> type RoundEventCallback func(ri *pb.RoundInfo, timedOut bool)
> ```

Showing that you can receive a full `RoundInfo` object for any round event
received by the client library on the network.

### Building the Library for iOS and Android

To set up gomobile for Android, install the NDK and pass the `-ndk` flag ` $ gomobile init`. Other repositories that use
gomobile for binding should include a shell script that creates the bindings. For iOS, gomobile must be run on an OS X
machine with Xcode installed.

Important reference info:

1. [Setting up gomobile and subcommands](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile)
2. [Reference cycles, type restrictions](https://pkg.go.dev/golang.org/x/mobile/cmd/gobind)

To clone and build:

```shell
# Go mobile install
$ go get golang.org/x/mobile/bind
$ go install golang.org/x/mobile/cmd/gomobile@latest
$ gomobile init... # Note this line will be different depending on sdk/target!

# Get and test code
$ git clone https://gitlab.com/elixxir/client.git client
$ cd client
$ go mod vendor
$ go mod tidy
$ go test ./...

# Android
$ gomobile bind -target android -androidapi 21 gitlab.com/elixxir/client/bindings

# iOS
$ gomobile bind -target ios gitlab.com/elixxir/client/bindings
$ zip -r iOS.zip Bindings.framework
```

You can verify that all symbols got bound by unzipping `bindings-sources.jar` and inspecting the resulting source files.

Every time you make a change to the client or bindings, you must rebuild the client bindings into a `.aar` or `iOS.zip`
to propagate those changes to the app. There's a script that runs gomobile for you in the `bindings-integration`
repository.

#### Android Maven Library

You can use the library published at the maven central repositories:

https://central.sonatype.com/search?q=xxdk

A gradle version 8.6 compatible `build.gradle` file is included that
allows you to build a maven-style library locally which you can
include in your package. This is used by us to sign and deploy to the
maven central repositories. You can use it as follows:

```shell
...
gomobile bind -v -target android -androidapi 21 gitlab.com/elixxir/client/v4/bindings
gradle publish && gradle genRepo
```

This will produce a local repository layout in the `build/repo` folder
and a zip file in the `build/distributions` folder. If you would like
to sign with your gpg key, you need to configure `gradle.properties` in
your user's `~/.gradle` folder, then change the version not to include
`-SNAPSHOT` at the end.


## Regenerate Protobuf File

First install the protobuf compiler or update by following the instructions in
[Installing Protocol Buffer Compiler](#installing-protocol-buffer-compiler)
below.

Use the following command to compile a protocol buffer.

```shell
protoc -I. -I../vendor --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative *.proto
```

* This command must be run from the directory containing the `.proto` file
  being compiled.
* The `-I` flag specifies where to find imports used by the `.proto` file and
  may need to be modified or removed to suit the .proto file being compiled.\
    * 💡 **Note:** Note: If you are importing a file from the vendor directory,
      ensure that you have the correct version by running `go mod vendor`.
* If there is more than one proto file in the directory, replace `*.proto` with
  the file’s name.
* If the `.proto` file does not use gRPC, then the `--go-grpc_out` and
  `--go-grpc_opt` can be excluded.



## Installing Protocol Buffer Compiler

This guide describes how to install the required dependencies to compile
`.proto` files to Go.

Before following the instructions below, be sure to remove all old versions of
`protoc`. If your previous protoc-gen-go file is not installed in your Go bin
directory, it will also need to be removed.

If you have followed this guide previously when installing `protoc` and need to
update, you can simply follow the instructions below. No uninstallation or
removal is necessary.

To compile a protocol buffer, you need the protocol buffer compiler `protoc`
along with two plugins `protoc-gen-go` and `protoc-gen-go-grpc`. Make sure you
use the correct versions as listed below.

|                      | Version | Download                                                            | Documentation                                                           |
|----------------------|--------:|---------------------------------------------------------------------|-------------------------------------------------------------------------|
| `protoc`             |  3.21.9 | https://github.com/protocolbuffers/protobuf/releases/tag/v3.21.9    | https://developers.google.com/protocol-buffers/docs/gotutorial          |
| `protoc-gen-go`      |  1.28.1 | https://github.com/protocolbuffers/protobuf-go/releases/tag/v1.28.1 | https://pkg.go.dev/google.golang.org/protobuf@v1.28.1/cmd/protoc-gen-go |
| `protoc-gen-go-grpc` |   1.2.0 | https://github.com/grpc/grpc-go/releases/tag/v1.2.0                 | https://pkg.go.dev/google.golang.org/grpc/cmd/protoc-gen-go-grpc        |

1. Download the correct release of `protoc` from the
   [release page](https://github.com/protocolbuffers/protobuf/releases) or use
   the link from the table above to get the download for your OS.

       wget https://github.com/protocolbuffers/protobuf/releases/download/v21.9/protoc-21.9-linux-x86_64.zip

2. Extract the files to a folder, such as `$HOME/.local`.

       unzip protoc-21.9-linux-x86_64.zip -d $HOME/.local

3. Add the selected directory to your environment’s `PATH` variable, make sure
   to include it in your `.profile` or `.bashrc` file. Also, include your go bin
   directory (`$GOPATH/bin` or `$GOBIN`) if it is not already included.

       export PATH="$PATH:$HOME/.local/bin:$GOPATH/bin"

   💡 **Note:** Make sure you update your configuration file once done with
   source `.profile`.

4. Now check that `protoc` is installed with the correct version by running the
   following command.

       protoc --version

   Which prints the current version

       libprotoc 3.21.9

5. Next, download `protoc-gen-go` and `protoc-gen-go-grpc` using the version
   found in the table above.

       go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
       go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

6. Check that `protoc-gen-go` is installed with the correct version.

       protoc-gen-go --version
       protoc-gen-go v1.28.1

7. Check that `protoc-gen-go-grpc` is installed with the correct version.

       protoc-gen-go-grpc --version
       protoc-gen-go-grpc 1.2.0

## Updating Valid Emoji List

The list of valid emojis should be updated once a year with each new Unicode
release. For more information, refer to
[generate/README.md](emoji/generate/README.md).

To run the generator from the repository root, run

```shell
go run ./emoji/generate/
```
