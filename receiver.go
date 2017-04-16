package ezmesh

/*
 * receiver.go
 * Receives messages
 * By J. Stuart McMurray
 * Created 20170411
 * Last Modified 20170413
 */

import (
	"github.com/weaveworks/mesh"
)

/* receiver handles received messages.  Specifically it calls it's Peer's
message handlers if they're set. */
type receiver struct {
	p *Peer
}

/* OnGossipUnicast is called by mesh when a unicast message has been
received. */
func (r *receiver) OnGossipUnicast(src mesh.PeerName, msg []byte) error {
	return r.handle(r.p.OnMessage, src, msg)
}

/* OnGossipBroadcast is called by mesh when a broadcast message has been
received. */
func (r *receiver) OnGossipBroadcast(
	src mesh.PeerName,
	msg []byte,
) (mesh.GossipData, error) {
	return gd(msg), r.handle(r.p.OnBroadcast, src, msg)
}

/* Since we don't keep state, this isn't useful */
func (r *receiver) Gossip() (complete mesh.GossipData) {
	return nil
}

/* Since we don't keep state, this isn't useful */
func (r *receiver) OnGossip(msg []byte) (delta mesh.GossipData, err error) {
	return delta, nil
}

/* handle handles an incoming message by extracting peer info and passing it
to f. */
func (r *receiver) handle(
	f func(*Peer, PeerName, []byte) error,
	n mesh.PeerName,
	msg []byte,
) error {
	/* Ignore the message if there's no handler installed */
	if nil == f {
		return nil
	}
	return f(r.p, PeerName(n), msg)
}

// gd turns a []byte into a mesh.GossipData
type gd []byte

func (g gd) Encode() [][]byte {
	return [][]byte{g}
}
func (g gd) Merge(o mesh.GossipData) mesh.GossipData {
	r := make([]byte, len(g))
	copy(r, g)
	/* Flatten o */
	for _, bs := range o.Encode() {
		r = append(r, bs...)
	}
	return gd(r)
}
