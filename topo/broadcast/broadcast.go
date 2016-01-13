package broadcast

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Gaboose/go-pubsub/topo"
	mux "github.com/jbenet/go-multicodec/mux"
)

type Msg struct {
	Id   string
	Data string
}

type msgInfo struct {
	msg    *Msg
	sender io.ReadWriteCloser
}

func (mi msgInfo) String() string {
	return fmt.Sprintf("%v", mi.msg)
}

type Broadcast struct {
	fanout      int
	protonet    topo.ProtoNet
	in          chan string
	out         chan string
	neighbCount chan int
	stop        chan bool

	cache      *ExpiringSet
	neighbsPri map[io.ReadWriteCloser]Peer
	neighbsSec map[io.ReadWriteCloser]bool
	neighbsmu  sync.RWMutex

	Str string
}

func (b *Broadcast) String() string {
	return b.Str
}

func New(fanout int, ttl time.Duration, protonet topo.ProtoNet) *Broadcast {
	return &Broadcast{
		cache:       NewExpiringSet(ttl),
		fanout:      fanout,
		protonet:    protonet,
		neighbCount: make(chan int, 1),
		neighbsPri:  map[io.ReadWriteCloser]Peer{},
		neighbsSec:  map[io.ReadWriteCloser]bool{},
	}
}

func (b *Broadcast) Start(peerSampler <-chan topo.Peer, backupSize int) {

	if b.stop != nil {
		panic(errors.New("Broadcast was already started"))
	}
	b.stop = make(chan bool)
	ready := make(chan bool)

	go func() {

		// set up channels

		b.in, b.out = make(chan string), make(chan string)
		outBuf := make(chan string)
		fromNeighbs, toNeighbs := make(chan msgInfo), make(chan msgInfo)
		newSecNeighbs := make(chan io.ReadWriteCloser)

		// start helper goroutines

		ln := b.protonet.Listen()
		go b.connAccepter(ln, newSecNeighbs)
		defer ln.Close()

		go b.broadcaster(toNeighbs)
		defer close(toNeighbs)

		go overflowBuffer(30, outBuf, b.out)
		defer close(outBuf)

		// start service logic goroutines

		go b.msgRouter(b.in, fromNeighbs, toNeighbs, outBuf)
		defer close(b.in)

		go b.neighbManager(backupSize, peerSampler, newSecNeighbs, fromNeighbs)

		ready <- true

		<-b.stop
	}()

	<-ready
}

func (b *Broadcast) Stop() {
	defer recover()
	close(b.stop)
}

func (b *Broadcast) In() chan<- string          { return b.in }
func (b *Broadcast) Out() <-chan string         { return b.out }
func (b *Broadcast) NeighbourCount() <-chan int { return b.neighbCount }

func (b *Broadcast) connect(p Peer, msgCh chan<- msgInfo, closedCh chan<- io.ReadWriteCloser) error {
	conn, err := b.protonet.Dial(p.Peer)
	if err == nil {
		p.conn = conn
		b.neighbsPri[conn] = p
		go b.msgAccepter(conn, msgCh, closedCh)
	}
	return err
}

func (b *Broadcast) outputNeighbCount() {
	select {
	case b.neighbCount <- len(b.neighbsPri) + len(b.neighbsSec):
	default:
	}
}

func (b *Broadcast) msgRouter(fromUser <-chan string,
	fromNeighbs <-chan msgInfo, toNeighbs chan<- msgInfo, toUser chan<- string) {
	for {
		select {
		case s, ok := <-fromUser:
			// Initiate a new broadcast

			if !ok {
				return
			}

			id := make([]byte, 32)
			rand.Read(id)
			m := Msg{Id: string(id), Data: s}

			b.cache.Add(m.Id)
			toNeighbs <- msgInfo{&m, nil}

		case mi := <-fromNeighbs:
			// Received a message from one of the neighbours.
			// Rebroadcast if we haven't seen it yet.

			if !b.cache.Has(mi.msg.Id) {
				b.cache.Add(mi.msg.Id)
				toNeighbs <- mi
				toUser <- mi.msg.Data
			}

		}
	}
}

func (b *Broadcast) neighbManager(backupSize int, peerSampler <-chan topo.Peer, newSecNeighbs <-chan io.ReadWriteCloser, fromNeighbs chan<- msgInfo) {
	connClosed := make(chan io.ReadWriteCloser)
	backup := make(OverflowSlice, 0, backupSize)

	for {
		select {
		case x := <-peerSampler:
			// keep our primary neighbour set filled with the youngest
			// peers received from this channel

			p := Peer{x, nil}

			b.neighbsmu.Lock()

			// the new peer can either be promoted to a neighbour or
			// added to the backup array

			oldest := NeighbourSet(b.neighbsPri).Oldest()
			if len(b.neighbsPri) < b.fanout+1 {
				// neighbour set not full yet
				b.connect(p, fromNeighbs, connClosed)

				// output the new number of neighbours
				b.outputNeighbCount()

			} else if p.GetBday() > oldest.GetBday() {
				// replace the oldest neighbour with this new peer
				// and store the demoted neighbour in the backup slice

				err := b.connect(p, fromNeighbs, connClosed)
				if err == nil {
					delete(b.neighbsPri, oldest.conn)
					oldest.conn.Close()
					backup = backup.Insert(*oldest)
				}

			} else {
				// none of our neighbours are older
				// store the new peer in the backup slice
				backup = backup.Insert(p)
			}

			b.neighbsmu.Unlock()

		case conn := <-newSecNeighbs:
			// Someone connected to us, store the connection so we can
			// broadcast back to them.
			// If they're behind a NAT, backstreams are the only way
			// for them to hear broadcasts.

			b.neighbsmu.Lock()
			b.neighbsSec[conn] = true
			b.outputNeighbCount()
			b.neighbsmu.Unlock()

			go b.msgAccepter(conn, fromNeighbs, connClosed)

		case conn := <-connClosed:
			// Remove the closed conection from our neighbour set.
			// If we initiated the connection (i.e. it's a primary
			// neighbour), try to replace it with one of the backup peers.

			b.neighbsmu.Lock()
			_, isPrimary := b.neighbsPri[conn]
			delete(b.neighbsPri, conn)
			delete(b.neighbsSec, conn)

			if isPrimary {
				// try to connect to one of our backup peers
				// starting from the youngest

				tail := backup.UntilFirst(func(p Peer) bool {
					err := b.connect(p, fromNeighbs, connClosed)
					return err == nil
				})

				// remove the peers we tried from backup
				copy(backup, tail)
				backup = backup[:len(tail)]
			}

			b.outputNeighbCount()
			b.neighbsmu.Unlock()

		case <-b.stop:
			b.neighbsmu.Lock()
			for conn, _ := range b.neighbsPri {
				conn.Close()
			}
			b.neighbsPri = nil
			for conn, _ := range b.neighbsSec {
				conn.Close()
			}
			b.neighbsSec = nil
			b.neighbsmu.Unlock()
			return
		}
	}

}

func (b *Broadcast) connAccepter(ln topo.Listener, out chan<- io.ReadWriteCloser) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		select {
		case out <- conn:
		case <-b.stop:
			conn.Close()
			return
		}
	}
}

func (b *Broadcast) msgAccepter(rwc io.ReadWriteCloser, out chan<- msgInfo, closed chan<- io.ReadWriteCloser) {
	mx := mux.StandardMux()
	for {
		m := &Msg{}
		err := mx.Decoder(rwc).Decode(m)
		if err == io.EOF || err == io.ErrClosedPipe {
			closed <- rwc
			return
		} else if err != nil {
			panic(err)
		}
		out <- msgInfo{m, rwc}
	}
}

func (b *Broadcast) broadcaster(in <-chan msgInfo) {
	mx := mux.StandardMux()
	for {
		mi, ok := <-in
		if !ok {
			return
		}

		buf := bytes.Buffer{}
		mx.Encoder(&buf).Encode(mi.msg)
		bts := buf.Bytes()

		b.neighbsmu.RLock()
		for conn, _ := range b.neighbsPri {
			if mi.sender != conn {
				go io.Copy(conn, bytes.NewReader(bts))
			}
		}
		for conn, _ := range b.neighbsSec {
			if mi.sender != conn {
				go io.Copy(conn, bytes.NewReader(bts))
			}
		}
		b.neighbsmu.RUnlock()
	}
}
