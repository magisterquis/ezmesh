ezmesh
======
[![GoDoc](https://godoc.org/github.com/magisterquis/ezmesh?status.svg)](https://godoc.org/github.com/magisterquis/ezmesh)
ezmesh is a lightweight, user-friendly mesh networking library built on top of
[github.com/weaveworks/mesh](https://github.com/weaveworks/mesh).

It provides a minimal interface focused on message-passing (and not maintaining
a shared state).  It handles the routing for messages to be sent peer-to-peer
or from a single peer to all the peers in a network.  There's no external
dependencies, as the underlying library has been vendored.

This library is in a fairly early stage of development.  Please don't use it in
production (or even critical development) environments.  The underlying library
is well-used, however, so it is unlikely to cause a total network meltdown.

I think.

Usage
-----
To join the mesh, call `ezmesh.New()`, and pass in a [`Config`](#config).  The
returned `Peer` represents the connection to the mesh network.

The `Peer`'s `Broadcast` and `Send` methods send a message to all other peers
in the network or a single peer, respectively.

If the `Peer`'s `OnBroadcast` and `OnMessage` fields are populated, they will
be called when the `Peer` receives a brodcast or unicast (i.e. peer-to-peer)
message, respectively.

Example
-------
```go
package main

import (
	"log"
	"net"
	"os"

	"github.com/magisterquis/ezmesh"
)

func main() {
	/* Make a PeerName, which uniquely identifies this Peer */
	pn, err := ezmesh.RandomPeerName()
	if nil != err {
		/* Handle error */
	}

	/* Initial Peer to which to connect, can be more than one */
	ip, err := net.ResolveTCPAddr("tcp", "192.168.0.1:3333")
	if nil != err {
		/* Handle error */
	}

	/* Configuration */
	conf := ezmesh.Config{
		Address:      "0.0.0.0",
		Port:         3333,
		Key:          []byte("kittens"),
		AutoConnect:  true,
		ConnLimit:    64,
		NickName:     "examplepeer",
		Name:         &pn,
		InitialPeers: []*net.TCPAddr{ip},
		OnMessage: func(
			p *ezmesh.Peer,
			src ezmesh.PeerName,
			message []byte,
		) error {
			nn, ok := p.NickNameFromPeerName(src)
			if !ok {
				nn = "unknown"
			}
			log.Printf("%v (%v): %q", nn, src, message)
			return nil
		},
		OnBroadcast: func(
			p *ezmesh.Peer,
			src ezmesh.PeerName,
			message []byte,
		) error {
			nn, ok := p.NickNameFromPeerName(src)
			if !ok {
				nn = "unknown"
			}
			log.Printf("Broadcast %v (%v): %q", nn, src, message)
			return nil
		},
	}

	/* Join the mesh network */
	peer, errs, err := ezmesh.New(
		conf,
		log.New(os.Stdout, "[meshdebug] ", log.LstdFlags),
	)
	if nil != err {
		/* Handle error */
	}
	if nil != errs {
		/* Handle connection errors */
	}

	/* Send a hello every so often */
	for {
		/* Say hello to everybody */
		peer.Broadcast([]byte("Hello!"))

		/* Send the controller a message, if he's there */
		if cn, ok := peer.PeerNameFromNickName("controller"); ok {
			peer.Send(cn, []byte("Hello, controller"))
		}
	}
}
```

Two example programs are provided in [./examples](examples) which together
form a minimal adminisration system (or a really chatty backdoor).  Please
don't use them in production, as they're not very robust or secure.

Config
------
Several parameters need to be set before joining the network, which are passed
to `New` in a struct.  Unfortunately, not all of the fields in the struct can
be left to default values.

### Address and Port
If the address is set, a listener will be started on the given address and port
which will accept connections from other peers.

### Key
A shared secret common to all members of the mesh network.

### AutoConnect
If true, connections will be attepmted to other members of the mesh network
besides the initial connections.  This improves robustness at the cost of
extra comms on the wire.

### ConnLimit
Limits the number of connections made.

### NickName
The human-friendly name for the `Peer` in the mesh network.  All members of the
mesh network have one, which can be retreived using the `PeerName` as a key
with `Peer`'s `NickNameFromPeerName()` method.

### Name
The `PeerName` for the `Peer`.  See the [`PeerName`](#PeerName) section.

### InitialPeers
A slice of `*net.TCPAddr`s to which to make the initial connections to the mesh
network.  The `[]error` returned from `New()` indicates any errors connecting
to these addresses.  If the length of the `[]error` is the number of the
initial peers, no connections were made.

### OnMessage
Callback function which will be called when a unicast (i.e. peer-to-peer)
message is received.  The first argument to the function is the local `Peer`,
i.e. not the peer in the mesh network which sent the message.  The sending
peer is identified by the second argument.  If `OnMessage` is `nil`, incoming
unicast messages will be discarded.

### OnBroadcast
Similar to OnMessage, but called for broadcast (i.e. peer-to-everybody)
messages.

PeerName
--------
Every member of the mesh network is identified by a unique `PeerName`, which is
an 8-byte number, of which 6 are used by the underlying library.

These may be generated from a MAC address with `PeerNameFromMACAddr()`,
randomly with `RandomPeerName()`, by hashing a string with
`PeerNameFromStringHash()`, or from a hex-digits-and-colon string (like a MAC
address) with `UnstringPeerName()`.  This is a somewhat awkward interface, and
is likely to be fixed in the future (without breaking existing code).

A mesh network peer's human-readable `NickName` may be retreived by calling
`Peer`'s `NickNameFromPeerName()` method, if the peer is known.  Likewise,
`Peer`'s `PeerNameFromNickName()` method does the reverse lookup.

It's not a bad idea to try to make sure `NickNameFromPeerName()`'s second
return value is true before sending a message with `Peer`'s `Send()` method,
especially right after joining, to make sure the message has a path to the
recipient.  There's an example of this in the example program
[`commander`](./examples/commander/commander.go).
