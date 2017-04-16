package main

/*
 * magent.go
 * ezmesh remote admin agent
 * By J. Stuart McMurray
 * Created 20170414
 * Last Modified 20170416
 */

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/magisterquis/ezmesh"
)

/* Process start time */
var startTime = time.Now()

func main() {
	var (
		laddr = flag.String(
			"l",
			"0.0.0.0:3000",
			"Listen `address`",
		)
		lport = flag.Uint(
			"p",
			3000,
			"Listen `port`",
		)
		key = flag.String(
			"k",
			"kittens",
			"Encryption `key`",
		)
		nick = flag.String(
			"n",
			"",
			"Peer `Nickname`",
		)
		initialPeers = flag.String(
			"i",
			"",
			"Initial connection address `list`",
		)
		bcint = flag.Duration(
			"b",
			10*time.Minute,
			"Hello broadcast `interval`",
		)
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %v [options]

Connects to a mesh network, using the addresses given with -i to find the first
peers.  After connecting, broadcasts a hello message periodically.  Messages
sent directly to this peer are assumed to be shell commands, which will be 
executed and the output of which sent back to the sender of the original
message.


The list of initial addresses to which to connect should be of the form
host:port,host:port,host:port

Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Work out initial peer list */
	ips := makeInitialPeers(*initialPeers)

	/* Make our mesh network peer */
	pn, err := ezmesh.RandomPeerName()
	if nil != err {
		log.Fatalf("Unable to make random peer name: %v", err)
	}
	peer, errs, err := ezmesh.New(
		ezmesh.Config{
			Address:      *laddr,
			Port:         uint16(*lport),
			Key:          []byte(*key),
			AutoConnect:  true,
			ConnLimit:    64,
			NickName:     *nick,
			Name:         &pn,
			InitialPeers: ips,
			OnMessage:    onMessage,
			OnBroadcast:  nil, /* Don't care about hellos */
		},
		log.New(
			os.Stdout,
			fmt.Sprintf("[%v|%v] ", pn, *nick),
			log.LstdFlags,
		),
	)
	if nil != err {
		log.Fatalf("Unable to make mesh peer: %v", err)
	}
	for _, err := range errs {
		log.Printf("Connection error: %v", err)
	}

	/* Every so often, say hello to the network */
	c := time.Tick(*bcint)
	for _ = range c {
		sayHello(peer)
	}
}

/* stringListToTCPAddr turns a list of comma-separated addresess into a slice
of *net.TCPAddrs.  It will terminate the program if there's not at least one
useable address. */
func makeInitialPeers(l string) []*net.TCPAddr {
	/* Split list into parts */
	parts := strings.Split(l, ",")
	/* Returnable array */
	ips := make([]*net.TCPAddr, 0, len(parts))
	/* Resolve each address */
	for _, p := range parts {
		/* Skip blank addresses, which happen because of repeated
		or trailing commas. */
		if 0 == len(p) {
			continue
		}
		/* Parse address */
		a, err := net.ResolveTCPAddr("tcp", p)
		if nil != err {
			log.Printf("Unable to resolve %q: %v", p, err)
			continue
		}
		/* Add it to the list of good addresses */
		ips = append(ips, a)
	}
	/* Die if we haven't any */
	if 0 == len(ips) {
		log.Fatalf("No good initial connection addresses found")
	}
	return ips
}

/* onMessage accepts a shell command, runs it, and sends back the result */
func onMessage(p *ezmesh.Peer, src ezmesh.PeerName, message []byte) error {
	/* Get the human-friendly name */
	nn, ok := p.NickNameFromPeerName(src)
	if !ok {
		nn = "?"
	}
	log.Printf("%v (%v) requests %q", src, nn, message)
	/* Make a command to run */
	var c *exec.Cmd
	if "windows" == runtime.GOOS {
		/* Blerg */
		c = exec.Command(
			"powershell.exe",
			"-NoP",
			"-NonI",
			"-W", "Hidden",
			"-Exec", "Bypass",
			"-C", string(message),
		)
	} else {
		c = exec.Command("/bin/sh", "-c", string(message))
	}
	/* Run it, get the output */
	o, err := c.CombinedOutput()
	/* Put the error message with the output */
	if nil != err {
		o = append(o, '\n')
		o = append(o, []byte(err.Error())...)
	}

	/* Send it back */
	if err := p.Send(src, o); nil != err {
		log.Printf(
			"Unable to send response to %v for %q: %v",
			src,
			message,
			err,
		)
		return err
	}

	return nil
}

/* sayHello broadcasts a little about us to the network */
func sayHello(p *ezmesh.Peer) {
	hn, err := os.Hostname()
	if nil != err {
		hn = fmt.Sprintf("Hostname:%v", err)
	}
	msg := fmt.Sprintf(
		"%v %v/%v %v",
		hn,
		runtime.GOOS,
		runtime.GOARCH,
		time.Since(startTime),
	)
	p.Broadcast([]byte(msg))
}
