# Server Initialization

These steps must first be performed in order to begin creating server objects of any variety.

### Make an api.Client Object

The api.Client object created here will be used for all types of api.Identity and server initialization.

1. Obtain the NDF

```go
ndfJson, err := api.DownloadAndVerifySignedNdfWithUrl(url, cert)
```

2. If not done in previous runs, create a new api.Client object in storage using ndfJson.
   `storageDir` and `password` may be customized.

Example:

```go
err := api.NewClient(ndfJson, "/clientStorage", []byte("testPassword"), "")
```

3. Login in order to obtain the api.Client object.
   `storageDir` and `password` may be customized, but must match the values provided to `NewClient()`.
   The result of `api.GetDefaultParams()` may also be freely modified according to your needs.

Example:

```go
client, err := api.Login("/clientStorage", []byte("testPassword"), api.GetDefaultParams())
```

4. Start the network follower. Timeout may be modified as needed.

Example:

```go
err := client.StartNetworkFollower(10*time.Second)
```

### Make an api.Identity Object

The api.Identity object created here will be used for all types of server initialization.
It requires an api.Client object.

Example:

```go
identity, err := api.MakeIdentity(client.GetRng(), client.GetStorage().GetE2EGroup())
```

# Building Servers

### Creating Connect-backed Servers

`receptionId`: the client ID that will be used for all incoming requests.
Derived from api.Identity object

`privKey`: the private key belonging to the receptionId.
Derived from api.Identity object

`rng`: from api.Client object

`grp`: from api.Client storage object

`net`: from api.Client object

`p`: customizable parameters for the server
Obtained and mutable via `connect.GetDefaultParams()`

Example:

```go
server, err := connect.NewServer(myIdentity.ID, myIdentity.DHKeyPrivate, client.GetRng(), 
	client.GetStorage().GetE2EGroup(), client.GetCmix(), connect.GetDefaultParams())
```

### Creating Single-backed Servers

`receptionId`: the client ID that will be used for all incoming requests.
Derived from api.Identity object

`privKey`: the private key belonging to the receptionId.
Derived from api.Identity object

`grp`: from api.Client storage object

`net`: from api.Client object

Example:

```go
server, err := single.NewServer(myIdentity.ID, myIdentity.DHKeyPrivate, 
	client.GetStorage().GetE2EGroup(), client.GetCmix())
```

# Adding Server Endpoints

Once you have a `server` object, you can begin adding Endpoints to process incoming client requests.
See documentation in restlike/types.go for more information.

Example:

```go
// Build a callback for the new endpoint
// The callback processes a restlike.Message and returns a restlike.Message response
cb := func(msg *Message) *Message {
    // Read the incoming restlike.Message and print its contents
    // NOTE: You may encode the msg.Contents in any way you like, as long as it matches on both sides.
    //       In this case, we're expecting a simple byte encoding.
    fmt.Printf("Incoming message: %s", string(msg.Content))
    // Return a friendly response to the incoming message 
    // NOTE: For responses, Content, Headers, and Error are the only meaningful fields
    return &restlike.Message{
		Content: []byte("Hello! Nice to meet you."),
		Headers: &restlike.Headers{
			Headers: nil,
			Version: 0,
		},
		Error: nil,
	}
}
// Add an endpoint that accepts 'restlike.Get' requests at the 'results' endpoint
server.GetEndpoints().Add("results", restlike.Get, cb)
```
