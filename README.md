# privategrity/client

[![pipeline status](https://gitlab.com/privategrity/client/badges/master/pipeline.svg)](https://gitlab.com/privategrity/client/commits/master)
[![coverage report](https://gitlab.com/privategrity/client/badges/master/coverage.svg)](https://gitlab.com/privategrity/client/commits/master)

This repo contains the Privategrity command-line client (used for integration
testing) and related libraries that facilitate making more full-featured
clientst for all platforms.

Running the Command Line Client
==

In project directory, run `go run main.go`

Required Args:

`-n <INT>`    : Number of nodes in the cMix network being connected to

`-i <INT>`    : User ID to log in as

Optional Args:

`-g <STRING>` : Address of the gateway to connect to (Required if not specified
in the config file)

`-d <INT>`    : ID of the user to send messages to

`-m <STRING>` : Message to be sent

`-v`          : Enables verbose logging

`-V`          : Show version information

`-f`          : String containing path of file to store the session into.
If not included it will use Ram Storage

`--noBlockingTransmission` : Disables transmission frequency limiting when 
specified

Example Configuration File
==

```yaml
logPath: "client.log"
numnodes : 3
sessionstore: "session.data"
textcolor: -1
gateways:
    - "gateway-0.prod.cmix.rip:11420"
    - "gateway-1.prod.cmix.rip:11420"
    - "gateway-2.prod.cmix.rip:11420"
```

##Project Structure

`api` package contains functions that clients written in Go should call to do
all of the main interactions with the client library.

`bindings` package exists for compatibility with Gomobile. All functions and
structs in the `bindings` package must be able to be bound with `$ gomobile bind`
or they will be unceremoniously removed. There are many requirements for 
this, and if you're writing bindings, you should check the `gomobile` 
documentation listed below.

In general, clients written in Go should use the `api` package and clients 
written in other languages should use the `bindings` package.

`bots` contains code for interacting with bots. If the amount of code required
to easily interact with a bot is reasonably small, it should go in this package.

`cmd` contains the command line client itself, including the dummy messaging
prototype that sends messages at a constant rate.

`crypto` contains code for encrypting and decrypting individual messages with
the client's part of the cipher. 

`globals` contains a few global variables. Avoid putting more things in here
without seriously considering the alternatives. Most important is the Log 
variable:

```go
globals.Log.ERROR.Println("this is an error")
```

Using this global Log variable allows external users of jww logging, like the 
console UI, to see and print log messages from the client library if they need
to, so please use globals.Log for all logging messages to make this behavior
work consistently.

If you think you can come up with a better design to deal with this problem, 
please go ahead and implement it. Anything that moves towards the globals 
package no longer existing is probably a win.

`io` contains functions for communicating between the client and the gateways.
It's also currently responsible for putting fragmented messages back together.

`parse` contains functions for serializing and deserializing various specialized
information into messages. This includes message types and fragmenting messages
that are too long.

`payment` deals with the wallet and payments, and keeping track of all related
data in non-volatile storage.

`switchboard` includes a structure that you can use to listen to incoming 
messages and dispatch them to the correct handlers.

`user` includes objects that deal with the user's identity and the session 
and session storage.

##Gomobile

We bind all exported symbols from the bindings package for use on mobile 
platforms. To set up Gomobile for Android, install the NDK and 
pass the -ndk flag to ` $ gomobile init`. Other repositories that use Gomobile
for binding should include a shell script that creates the bindings.

###Recommended Reading for Gomobile

https://godoc.org/golang.org/x/mobile/cmd/gomobile (setup and available 
subcommands)

https://godoc.org/golang.org/x/mobile/cmd/gobind (reference cycles, type 
restrictions)

Currently we aren't using reverse bindings, i.e. calling mobile from Go.

###Testing Bindings via Gomobile

The separate `bindings-integration` repository exists to make it easier to 
automatically test bindings. Writing instrumented tests from Android allows 
you to create black-box tests that also prove that all the methods you think 
are getting bound are indeed bound, rather than silently getting skipped.

You can also verify that all symbols got bound by unzipping `bindings-sources.jar`
and inspecting the resulting source files.

Every time you make a change to the client or bindings, you must rebuild the 
client bindings into a .aar to propagate those changes to the app. There's a 
script that runs gomobile for you in the `bindings-integration` repository.
