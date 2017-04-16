package ezmesh

/*
 * peername.go
 * Easy peer-naming scheme
 * By J. Stuart McMurray
 * Created 20170413
 * Last Modified 20170413
 */

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"net"

	"github.com/weaveworks/mesh"
)

// PeerName uniquely identifies a peer in the mesh network.  It is common to
// use the peer's network adapter's MAC address.  Only the lower six bytes are
// used.
type PeerName uint64

// RandomPeerName generates a random PeerName.
func RandomPeerName() (PeerName, error) {
	/* Get random bytes */
	b := make([]byte, 6)
	if _, err := rand.Read(b); nil != err {
		return PeerName(0), err
	}
	return PeerName(mesh.PeerNameFromBin(b)), nil
}

// PeerNameFromStringHash transforms s into a PeerName using the first 8 bytes
// of the SHA512 hash of the string.
func PeerNameFromStringHash(s string) PeerName {
	h := sha512.Sum512([]byte(s))
	return PeerName(binary.LittleEndian.Uint64(h[:8]))
}

// PeerNameFromMACAddr transforms a MAC address (or any other colon-separated
// series of hex-encoded bytes) into a PeerName.  Contiguous 0x00's may be
// elided, as in IPv6 addresses.  At most 6 colon-separated hex-encoded bytes
// may be turned into a PeerName.  This is a wrapper around
// mesh.PeerNameFromString.
func PeerNameFromMACAddr(a string) (PeerName, error) {
	pn, err := mesh.PeerNameFromString(a)
	return PeerName(pn), err
}

// PeerNameFromHardwareAddr return a PeerName made from the given hardware
// address.
func PeerNameFromHardwareAddr(h net.HardwareAddr) (PeerName, error) {
	if 8 == len(h) {
		return PeerName(binary.BigEndian.Uint64(h)), nil
	} else if 8 > len(h) {
		return PeerNameFromMACAddr(h.String())
	} else {
		return PeerNameFromStringHash(h.String()), nil
	}
}

// String encodes the PeerName as a string similar to a MAC address.
func (n PeerName) String() string {
	return mesh.PeerName(n).String()
}

// UnStringPeerName reverses the output of PeerName's String() method.  It
// accepts a hex-digits-and-colon name.
func UnStringPeerName(s string) (PeerName, error) {
	pn, err := mesh.PeerNameFromString(s)
	return PeerName(pn), err
}
