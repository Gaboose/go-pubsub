# P2P Publish/Subscribe Network

work in progress

#### current major limitations
* not optimized for multiple topics
* only communicates within a local area network

## Install

```bash
export GO15VENDOREXPERIMENT=1
go get -u github.com/Gaboose/go-pubsub/cmd/pubsub
```

## Usage

```bash
Usage: pubsub [<flags>] <command> [ -help | <args> ] - p2p pubsub network

COMMANDS:
    daemon              Run a network-connected pubsub node
    pub <topic> <msg>   Publish a message
    sub <topic>         Listen for and receive messages

FLAGS:
    -apiport    int - Port of the daemon API to connect to

Use 'pubsub <command> -help' for more information about a command.
```

## Code Structure

Every subpackage in package `go-pubsub/topo` represents a layer of network topology and protocol. Package `go-pubsub/net` combines them to form a functional network and exposes functions to a daemon in `go-pubsub/cmd`.

Topology packages don't use the standard `net` library. Instead they use a `ProtoNet` object defined in `go-pubsub/gway` to serve as a gateway. `ProtoNet` provides Dial and Listen methods like the standard `net` library, but abstracts away the internet transport layer (tcp, websocket, etc.) and preselects the same "protocol" (or topology layer) on the remote node as  the local one is in. This way the logic of a topology layer is guaranteed to be contained within its package and may be tested or reused as a singular unit.

## Topos

#### `go-pubsub/topo/cyclon`

[Journal Article](https://www.mendeley.com/catalog/cyclon-inexpensive-membership-management-unstructured-p2p-overlays)

Cyclon is an unstructured peer sampling layer. It holds a number of peer profiles. At constant time intervals it exchanges a random portion of them with one of the peers. Locally it provides other topos with a channel, which outputs profiles of uniformly random peers in the Cyclon layer.

#### `go-pubsub/topo/broadcast`

Broadcast is a flooding layer. It accepts a channel of peer profiles, connects to a small number of them and relays messages, which it hasn't seen recently. Locally it provides external packages with in and out channels to send and receive messages.

### To do:

* `go-pubsub/topo/vicinity` [Journal Article](https://www.mendeley.com/research/vicinity-pinch-randomness-brings-structure-1)
* `go-pubsub/topo/rings` as described in [Poldercast Article](https://www.mendeley.com/research/poldercast-fast-robust-scalable-architecture-p2p-topicbased-pubsub)
