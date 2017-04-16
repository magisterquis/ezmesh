// Package ezmesh wraps github.com/weaveworks/mesh into a more user-friendly
// library
package ezmesh

/*
 * ezmesh.go
 * Wraps github.com/weaveworks/mesh for ease of use
 * By J. Stuart McMurray
 * Created 20170410
 * Last Modified 20170413
 */

import (
	"fmt"
	"net"

	"github.com/weaveworks/mesh"
)

// PROTOVERSION is the protocol version used by this package.
const PROTOVERSION = 2

// Peer represents the local peer in the mesh network.
type Peer struct {
	// OnMessage is called when a unicast (peer-to-peer) message is
	// received, if it is not nil.  A pointer to this struct is passed in.
	OnMessage func(p *Peer, src PeerName, message []byte) error

	// OnBroadcast is called when a broadcast (peer-to-everybody) message
	// is received, if it is not nil.  A pointer to this struct is passed
	// in.
	OnBroadcast func(p *Peer, src PeerName, message []byte) error

	// The following provide access to the underlying mesh structures.
	// Please see https://godoc.org/github.com/weaveworks/mesh for more
	// information.
	TX     mesh.Gossip
	RX     mesh.Gossiper
	Router *mesh.Router
}

// Config contains parameters needed to connect to the mesh network.
type Config struct {
	// Address and Port specify the IP address and port on which to listen
	// for connections from other peers.  The address may be the empty
	// string, in which case a listener won't be started.
	Address string
	Port    uint16

	// Key provides an encryption key which must be the same for all peers.
	Key []byte

	// AutoConnect controls whether or not connections are automatically
	// attempted to other peers.
	AutoConnect bool

	// ConnLimit specifies the maximum number of peer connections.  A value
	// <= 0 allows unlimited connections.
	ConnLimit int

	// Nickname is the human-friendly name by which this peer will be known.
	NickName string

	// Name is the 8-byte name which represents this peer.  It is common
	// to use a MAC address (via PeerNameFromHardwareAddr).  If Name is
	// nil, it will be generated from NickName.
	Name *PeerName

	// InitialPeers is a slice of addresses containing other peers to which
	// to connect.  This slice may be empty.  More peers may be added with
	// Peer's Connect method.
	InitialPeers []*net.TCPAddr

	// OnMessage sets the OnMessage field in the generated Peer
	OnMessage func(p *Peer, src PeerName, message []byte) error

	// OnBroadcast sets the OnBroadcast field in the generated Peer
	OnBroadcast func(p *Peer, src PeerName, message []byte) error
}

// Logger is a simple logging interface.  It can be satisfied by *log.Logger.
type Logger interface {
	Printf(format string, args ...interface{})
}

// New creates a new peer joined to the mesh network.  The returned slice of
// errors will only contain errors from the initial connections.  If there
// are fewer errors than initial peers, then at least one connection succeeded.
// Even if all of the initial connections failed, the returned *Peer's Connect
// method may be used to attempt more connections.
//
// The returned error is non-nil if the returned peer is unusable.
//
// Unfortunately, due to the underlying library, there's no good way to know if
// listening for connections succeeded.
func New(config Config, l Logger) (*Peer, []error, error) {
	/* Make sure we have a nickname */
	if "" == config.NickName {
		return nil, nil, fmt.Errorf("no nickname")
	}
	/* Make sure we have a Name */
	if nil == config.Name {
		pn := PeerNameFromStringHash(config.NickName)
		config.Name = &pn
	}
	/* Make sure we don't have a negative connection limit */
	if 0 > config.ConnLimit {
		config.ConnLimit = 0
	}

	/* Peer to return */
	peer := &Peer{}
	router := mesh.NewRouter(
		mesh.Config{
			Host:               config.Address,
			Port:               int(config.Port),
			ProtocolMinVersion: PROTOVERSION,
			Password:           config.Key,
			ConnLimit:          int(config.ConnLimit),
			PeerDiscovery:      config.AutoConnect,
			TrustedSubnets:     []*net.IPNet{},
		},
		mesh.PeerName(*config.Name),
		config.NickName,
		mesh.NullOverlay{},
		l,
	)
	peer.Router = router
	peer.OnMessage = config.OnMessage
	peer.OnBroadcast = config.OnBroadcast

	/* Sending and receiving structs.  In theory we could have a bunch of
	channels, or maybe an interface to make more or something. */
	rx := &receiver{peer}
	tx := router.NewGossip("defaultchannel", rx)
	peer.RX = rx
	peer.TX = tx

	/* Start the listener if we're meant to */
	if "" != config.Address {
		peer.Router.Start()
	}

	/* Connect to the initial peers */
	errs := peer.Connect(config.InitialPeers)

	return peer, errs, nil
}

// Broadcast sends a message to every peer in the mesh network.
func (p *Peer) Broadcast(message []byte) {
	p.TX.GossipBroadcast(gd(message))
}

// Send sends the message to the specified peer.
func (p *Peer) Send(dst PeerName, message []byte) error {
	return p.TX.GossipUnicast(mesh.PeerName(dst), message)
}

// Connect makes connections to the given addresses.  Please see New for the
// meaning of the returned error slice.
func (p *Peer) Connect(addrs []*net.TCPAddr) []error {
	if nil == addrs {
		return nil
	}
	/* Turn addresses into strings */
	ips := make([]string, len(addrs))
	for i, p := range addrs {
		ips[i] = p.String()
	}
	return p.Router.ConnectionMaker.InitiateConnections(ips, false)
}

// PeerNameFromNickName returns the PeerName of a peer in the mesh network from
// with the given NickName.  It is an O(n) operation over the set of known
// peers.  If the NickName is not found, ok will be false.
func (p *Peer) PeerNameFromNickName(nickname string) (name PeerName, ok bool) {
	for _, pd := range p.Router.Peers.Descriptions() {
		if pd.NickName == nickname {
			return PeerName(pd.Name), true
		}
	}
	return PeerName(0), false
}

// NickNameFromPeerName returns the NickName of the peer in the peer mesh with
// the given PeerName.  It is an O(n) operation over the set of known peers.
// If the name is not found, ok will be false.
func (p *Peer) NickNameFromPeerName(name PeerName) (nickname string, ok bool) {
	for _, pd := range p.Router.Peers.Descriptions() {
		if PeerName(pd.Name) == name {
			return pd.NickName, true
		}
	}
	return "", false
}
