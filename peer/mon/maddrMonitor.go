package addrMon

import (
	"time"

	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
)

type Resource ma.Multiaddr

type Status struct {
	Alive bool
}

type Update struct {
	Res Resource
	St  Status
}

type IterRequest struct {
	// Can only use AliveIterator here for now
	ItFunc func(map[Resource]Status) IterResponse
	Ch     chan<- IterResponse
}

type IterResponse struct {
	Out  <-chan Resource
	Done chan<- bool // Remember to close this after using the iterator
}

func Start() (chan<- Resource, chan<- IterRequest, <-chan Update) {
	// This creates 3 long running goroutines:
	// - 'poller' probes addresses (or Resources) for their liveliness,
	//   latency, etc.
	// - 'statusMonitor' tracks address status
	// - 'loopback' provides poller with work in specific time intervals
	//
	// External packages should insert new addresses via the 'pending'
	// channel, request iterators via 'requests' and listen for updates via
	// 'updatesSnk'.

	pending, complete := make(chan Resource), make(chan Resource)

	// poller --updates-\--> stateMonitor
	//                   \-> (external)
	//
	// (external) --requests--> stateMonitor
	updates, requests := stateMonitor()
	updates, updatesSnk := fanout(updates)

	//                    (external)-\
	//                                \
	// poller --complete--> loopback --pending--> poller
	go poller(pending, complete, updates)
	go loopback(complete, pending)

	return pending, requests, updatesSnk
}

func AliveIterator(statuses map[Resource]Status) IterResponse {
	out := make(chan Resource)

	// Close 'done' channel after using the
	// iterator to dispose it gracefully
	done := make(chan bool)
	go func() {
		for r := range statuses {
			select {
			case out <- r:
			case <-done:
				return
			}
		}
		close(out)
	}()
	return IterResponse{Out: out, Done: done}
}

func stateMonitor() (chan<- Update, chan<- IterRequest) {
	updates := make(chan Update)
	reqs := make(chan IterRequest)
	statuses := make(map[Resource]Status)
	go func() {
		for {
			select {
			case u := <-updates:
				if !u.St.Alive {
					delete(statuses, u.Res)
				} else {
					statuses[u.Res] = u.St
				}
			case r := <-reqs:
				//guard from blocking on send
				go func() {
					// build and send back the iterator
					r.Ch <- r.ItFunc(statuses)
				}()
			}
		}
	}()
	return updates, reqs
}

func poller(in, out chan Resource, updates chan<- Update) {
	for r := range in {
		c, err := manet.Dial(r)
		if err != nil {
			// If address is dead, drop it out of circulation
			updates <- Update{Res: r, St: Status{Alive: false}}
		} else {
			c.Close()
			updates <- Update{Res: r, St: Status{Alive: true}}
			out <- r
		}
	}
}

func loopback(in, out chan Resource) {
	for {
		time.Sleep(time.Second)
		in <- <-out
	}
}

func fanout(snk1 chan<- Update) (chan<- Update, <-chan Update) {
	src, snk2 := make(chan Update), make(chan Update)
	go func() {
		for {
			u := <-src
			snk1 <- u
			snk2 <- u
		}
	}()
	return src, snk2
}
