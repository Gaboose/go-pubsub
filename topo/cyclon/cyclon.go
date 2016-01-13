package cyclon

// Uniformly random peer sampling service.
// "Cyclon" is implemented following this journal article:
//
// Voulgaris, S., Gavidia, D. & Van Steen, M., 2005.
// CYCLON: Inexpensive membership management for unstructured P2P overlays.
// Journal of Network and Systems Management, 13(2), p.197-216.

import (
	"errors"
	//"fmt"
	"net/rpc"
	"sync"
	"time"

	"github.com/Gaboose/go-pubsub/topo"
)

// Names of topo.Peer parameters
const age = "age"
const bday = "bday"

type Cyclon struct {
	me         topo.Peer
	cachesize  int
	shuflen    int
	neighbs    PeerSet // our "neighbour set" or "cache"
	neighbsmu  sync.RWMutex
	protonet   topo.ProtoNet
	serviceAge int64
	out        chan topo.Peer
	outBuf     chan topo.Peer
	stop       chan bool
}

func New(me topo.Peer, cachesize, shuflen int, protonet topo.ProtoNet) *Cyclon {
	me.Put(age, 0)
	return &Cyclon{
		me:        me,
		cachesize: cachesize,
		shuflen:   shuflen,
		neighbs:   make(PeerSet),
		protonet:  protonet,
	}
}

func (c *Cyclon) Start(interval time.Duration) {
	if c.stop != nil {
		panic(errors.New("Cyclon is already running"))
	}
	c.stop = make(chan bool)

	// Start RPC server
	var stop [2]chan bool
	stop[0] = CyclonRPC{c}.serve()

	// Start periodic shuffling
	if interval > 0 {
		stop[1] = c.tick(interval)
	}

	// Start output buffer
	c.outBuf, c.out = make(chan topo.Peer), make(chan topo.Peer)
	go overflowBuffer(c.cachesize, c.outBuf, c.out)

	go func() {
		<-c.stop
		c.stop = nil

		close(stop[0])
		if stop[1] != nil {
			close(stop[1])
		}
		close(c.outBuf)
	}()
}

func (c *Cyclon) Stop() {
	close(c.stop)
}

func (c *Cyclon) tick(interval time.Duration) chan bool {
	stop := make(chan bool)
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.Shuffle()
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()
	return stop
}

// Add adds peers to the cache. You'll need to call this to jumpstart Cyclon.
func (c *Cyclon) Add(peers ...topo.Peer) {
	// TODO: Use random walks instead of shuffling to jumpstart,
	// like the paper specifies.
	c.neighbsmu.Lock()
	for _, p := range peers {
		p.Put(age, 0)
		c.neighbs[p.Id()] = p
	}
	c.neighbsmu.Unlock()
}

// Out channel constantly sends new peers from the Cyclon network
// as they're discovered.
func (c *Cyclon) Out() <-chan topo.Peer {
	return c.out
}

// Shuffle initiates a random exchange of peer profiles with a neighbour of
// greatest age.
func (c *Cyclon) Shuffle() {
	c.neighbsmu.Lock()
	if len(c.neighbs) == 0 {
		c.neighbsmu.Unlock()
		return
	}

	// Increase the age of all neighbours and our service
	for _, p := range c.neighbs {
		p.Put(age, p.Get(age).(int)+1)
	}
	c.serviceAge++

	// Pop the neighbour that we're going to shuffle with
	q := c.neighbs.PopOldest()

	// Construct the offer. This doesn't remove entries from c.neighbs
	offer := c.neighbs.Sample(c.shuflen - 1)

	c.neighbsmu.Unlock()

	// Calling another cyclon over the network can take a while
	// so we keep our cache unlocked while doing this.
	var answer []topo.Peer
	conn, err := c.protonet.Dial(q)
	if err == nil {
		cl := rpc.NewClient(conn)
		cl.Call("CyclonRPC.HandleShuffle", append(offer, c.me), &answer)
		conn.Close()
	}

	c.neighbsmu.Lock()
	c.updateCache(answer, offer)
	c.neighbsmu.Unlock()
}

type CyclonRPC struct{ c *Cyclon }

func (r CyclonRPC) HandleShuffle(offer []topo.Peer, answer *[]topo.Peer) error {
	c := r.c
	c.neighbsmu.Lock()
	*answer = c.neighbs.Sample(c.shuflen)
	c.updateCache(offer, *answer)
	c.neighbsmu.Unlock()
	return nil
}

func (r CyclonRPC) serve() chan bool {

	// Serve protonet connections, so that this server is only available
	// to dialers running the Cyclon protocol.
	ln := r.c.protonet.Listen()

	stop := make(chan bool)
	go func() {
		<-stop
		ln.Close()
	}()

	// Run an rpc server to handle shuffle calls
	s := rpc.NewServer()
	s.Register(r)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}
			go s.ServeConn(conn)
		}
	}()

	return stop
}

func (c Cyclon) updateCache(new, old []topo.Peer) {
	// Filter out entries that are already in the cache or equal to c.me
	for i := 0; i < len(new); i++ {
		if _, has := c.neighbs[new[i].Id()]; has || c.me.Id() == new[i].Id() {
			new[i] = new[len(new)-1]
			new = new[:len(new)-1]
			i--
		}
	}

	// Send the new peers out without blocking
	if c.outBuf != nil {
		for _, p := range new {
			p.Put(bday, c.serviceAge-int64(p.Get(age).(int)))
			c.outBuf <- p
		}
	}

	// Push new peers to our neighbour set up until it's full
	for len(c.neighbs) < c.cachesize && len(new) > 0 {
		c.neighbs[new[0].Id()] = new[0]
		new = new[1:]
	}

	// Write the remaining answer on top of peers we sent out
	for len(old) > 0 && len(new) > 0 {
		// Check if we still have the old peers, so we're sure
		// we're replacing entries rather than pushing new ones.
		if _, has := c.neighbs[old[0].Id()]; has {
			delete(c.neighbs, old[0].Id())
			c.neighbs[new[0].Id()] = new[0]
			new = new[1:]
		}
		old = old[1:]
	}
}
