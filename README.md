# elixxir/client

[![pipeline status](https://gitlab.com/elixxir/client/badges/master/pipeline.svg)](https://gitlab.com/elixxir/client/commits/master)
[![coverage report](https://gitlab.com/elixxir/client/badges/master/coverage.svg)](https://gitlab.com/elixxir/client/commits/master)

This repo contains the Elixxir command-line client (used for integration
testing) and related libraries that facilitate making more full-featured
clients for all platforms.

##Running the Command Line Client

First, make sure dependencies are installed into the vendor folder by running
`glide up`. Then, in the project directory, run `go run main.go`.

If what you're working on requires you to change other repos, you can remove
the other repo from the vendor folder and Go's build tools will look for those
packages in your Go path instead. Knowing which dependencies to remove can be
really helpful if you're changing a lot of repos at once.

If glide isn't working and you don't know why, try removing glide.lock and
~/.glide to brutally cleanse the cache.


Mutually exclusive (almost) required args:

|Long flag|Short flag|Effect|Example|
|---|---|---|---|
|--userid|-i|ID of precanned user to use|-i 5|
|--regcode|-e|Registration code to use for logging in a new user|-e AAAA|

The above args are mutually exclusive and are not fully required.

For example, to login as canned user 18, use `-i 18` and any registration code specified with `-e` will be ignored.
To login as a new user, `-i` MUST not be specified, and `-e` will be the registration code to be used.

NOTE: There is a third way of starting the client, which ONLY works without specifying any of the above args.
This will internally ignore the registration address, if specified, and will do registration directly on the Nodes
only.

Optional args:

|Long flag|Short flag|Effect|Example|
|---|---|---|---|
|--message|-m|Message to send|-m "top of the morning"|
|--messageTimeout|-t|The number of seconds to wait for 'waitForMessages' messages to arrive (default 45)|-t 42|
|--ndf|-n|Path to the network definition JSON file (default "ndf.json")| -n "ndf.json"|
|--SearchForUser|-s|Sets the email to search for to find a user with user discovery| -s "david@chaum.com|
|--dest64| |Sets the destination user id encoded in base 64| --dest64 "yCvV6AsEK3l+45Gn4awBJ4lpb+hT2sO6yzxjeraRor0="|
|--destid|-d|ID to send message to| -d 69|
|--email|-E|Email to register for User Discovery| -e "david@chaum.com"|
|--end2end| |Send messages with E2E encryption to destination user. Must have found each other via UDB first| -end2end|
|--help| |help for client| --help|
|--keyParams| |Define key generation parameters. Pass values in comma separated list in the following order: MinKeys,MaxKeys,NumRekeys,TTLScalar,MinNumKeys| |
|--ndfPubKey|-p|Path to the public key for the network definition JSON file|
|--nick| |Nickname to register for User Discovery (default "Default")| --nick "zezima"|
|--noBlockingTransmission| |Sets if transmitting messages blocks or not.  Defaults to true if unset.|--noBlockingTransmission|
|--noTLS| |Set to ignore TLS. Connections will fail if the network requires TLS. For debugging|--noTLS|
|--privateKey| |The path for a PEM encoded private key which will be used to create the user|--privateKey "key.pem"|
|--rateLimiting| |Sets the amount of time, in ms, that the client waits between sending messages.  set to zero to disable.  Automatically disabled if 'blockingTransmission' is false (default 1000)| --rateLimiting 100|
|--regcode string|-r|Registration Code with the registration server |--regcode "AAAA"|
|--sessionfile|-f|Passes a file path for loading a session.  If the file doesnt exist the code will register the user and store it there.  If not passed the session will be stored to ram and lost when the cli finishes| -s "user.session"|
|--skipNDFVerification| |Specifies if the NDF should be loaded without the signature|--skipNDFVerification|
|--userid|-i|ID to sign in as. Does not register, must be an available precanned user |-i 32|
|--verbose|-v|Verbose mode for debugging|-v|
|--version|-V|Show the client version information|-V|
|--waitForMessages|-w|Denotes the number of messages the client should receive before closing (default 1)|-w 7|

Runs a client for cMix anonymous communication platform

Use "client [command] --help" for more information about a command.



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

globals.Log.ERROR.Println("this is an error")

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
