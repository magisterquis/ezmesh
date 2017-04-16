Example Programs
================
These two programs demonstrate how to use ezmesh.  Please do not use them for
illegal purposes.

Agent
-----
This is a small program which sits on a host and announces its presence every
so often.  If it gets a unicast message, it'll run it as if it were a command
and send the output back.  This is not secure.

Commander
---------
Operates in two modes.

It can sit and listen for broadcast messages, which it prints to stdout.

Alternatively, it can send a message to another peer on the mesh network and
print a response from that peer.
