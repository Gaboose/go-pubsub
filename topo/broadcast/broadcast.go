package broadcast

import (
	"crypto/rand"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/Gaboose/go-pubsub-planet/topo"
	mux "github.com/jbenet/go-multicodec/mux"
)

// Name of a parameter of topo.Peer
const bday = "bday"

type Broadcast struct {
	me          interface{}
	fanout      int
	protnet     topo.ProtNet
	in          chan string
	out         chan string
	neighbCount chan int

	cache      *ExpiringSet
	neighbsPri map[io.ReadWriteCloser]int64
	neighbsSec map[io.ReadWriteCloser]bool
	neighbsmu  sync.RWMutex
}

type Msg struct {
	Id       string
	Data     string
}

type msgInfo struct {
	msg &Msg
	sender io.ReadWriteCloser
}

func New(me interface{}, fanout int, ttl time.Duration, protnet topo.ProtNet) *Broadcast {
	return &Broadcast{
		me:         me,
		cache:      NewExpiringSet(ttl),
		fanout:     fanout,
		protnet:    protnet,
		neighbsPri: map[io.ReadWriteCloser]int64{},
		neighbsSec: map[io.ReadWriteCloser]bool{},
	}
}

func (b *Broadcast) Start(peerSampler <-chan topo.Peer, backupSize int) {

	// set channels up and start helper goroutines

	newSecNeighbs := make(chan io.ReadWriteCloser)
	ln := b.protnet.Listen()
	go b.connAccepter(ln, newSecNeighbs)

	fromNeighbs, toNeighbs := make(chan msgInfo), make(chan msgInfo)
	go b.broadcaster(toNeighbs)

	b.in, b.out = make(chan string), make(chan string)

	outBuf := make(chan string)
	go overflowBuffer(30, outBuf, b.out)

	// start logic goroutines

	// message router
	go func() {
		for {
			select {
			case s := <-b.in:
				// Initiate a new broadcast

				id := make([]byte, 32)
				rand.Read(id)
				m := Msg{Id: string(id), Data: s}

				b.cache.Add(m.Id)
				toNeighbs <- msgInfo{&m, nil}

			case mi := <-fromNeighbs:
				// Received a message from one of the neighbours.
				// Rebroadcast if we haven't seen it yet.

				fmt.Printf("%v: got %v, has: %v\n", b.me, mi, b.cache.Has(mi.msg.Id))
				if !b.cache.Has(mi.msg.Id) {
					b.cache.Add(mi.msg.Id)
					toNeighbs <- mi
					outBuf <- mi.msg.Data
				}

			}
		}
	}()

	// neighbour manager
	go func() {
		connClosed := make(chan io.Reader)
		backup := make([]topo.Peer, 0, backupSize)

		for {
			select {
			case p := <-peerSampler:
				// keep our primary neighbour set filled with the youngest
				// peers received from this channel

				bd := p.Get(bday).(int64)

				b.neighbsmu.Lock()

				// if our neighbour set is full, find the oldest neighbour
				var oldest io.ReadWriteCloser
				if len(b.neighbsPri) >= b.fanout+1 {
					var min int64
					for n, i := range b.neighbsPri {
						if i < min {
							min = i
							oldest = n
						}
					}

					// all of our neighbours are younger?
					// insert the new peer into the sorted backup array
					if bd < min {
						i := sort.Search(len(backup), func(i int) bool {
							return backup[i].Get(bday).(int64) <= bd
						})
						if i < cap(backup) {
							if len(backup) < cap(backup) {
								backup = backup[:len(backup)+1]
							}
							copy(backup[i+1:], backup[i:])
							backup[i] = p
						}
						b.neighbsmu.Unlock()
						break
					}
				}

				// if we can connect to the new peer
				// remove the oldest neighbour
				if err := b.connect(p, fromNeighbs, connClosed); err != nil {
					break
				}
				delete(b.neighbsPri, oldest)
				oldest.Close()

				b.outputNeighbCount()
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

			case c := <-connClosed:
				// Remove the closed conection from our neighbour set.
				// If we initiated the connection (i.e. it's a primary
				// neighbour), try to replace it with one of the backup peers.

				conn := c.(io.ReadWriteCloser)

				b.neighbsmu.Lock()
				delete(b.neighbsPri, conn)
				delete(b.neighbsSec, conn)

				if _, has := b.neighbsPri[conn]; has {
					// try to connect to one of our backup peers
					// starting from the youngest
					newbackup := backup
					for len(newbackup) > 0 {
						p := newbackup[0]
						newbackup = newbackup[1:]
						err := b.connect(p, fromNeighbs, connClosed)
						if err == nil {
							break
						}
					}

					// remove the peers we tried from backup
					copy(backup, newbackup)
					backup = backup[:len(newbackup)]
				}

				b.outputNeighbCount()
				b.neighbsmu.Unlock()
			}
		}
	}()

}

func (b *Broadcast) In() chan<- string          { return b.in }
func (b *Broadcast) Out() <-chan string         { return b.out }
func (b *Broadcast) NeighbourCount() <-chan int { return b.neighbCount }

func (b *Broadcast) connect(p topo.Peer, msgCh chan<- *Msg, closedCh chan<- io.Reader) error {
	conn, err := b.protnet.Dial(p)
	if err == nil {
		b.neighbsPri[conn] = p.Get(bday).(int64)
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

func (b *Broadcast) connAccepter(ln topo.Listener, out chan<- io.ReadWriteCloser) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		out <- conn
	}
}

func (b *Broadcast) msgAccepter(rwc io.ReadWriteCloser, out chan<- *Msg, closed chan<- io.ReadWriteCloser) {
	mx := mux.StandardMux()
	for {
		m := &Msg{}
		if err := mx.Decoder(rwc).Decode(m); err == io.EOF {
			closed <- rwc
			return
		}
		out <- msgInfo{m, rwc)
	}
}

func (b *Broadcast) broadcaster(in <-chan msgInfo) {
	mx := mux.StandardMux()
	for {
		mi := <-in
		buf := bytes.Buffer{}
		mx.Encoder(&buf).Encode(mi.msg)
		bts := buf.Bytes()

		b.neighbsmu.RLock()
		for conn, _ := range b.neighbsPri {
			if msgInfo.sender != conn {
				io.Copy(conn, bytes.NewReader(bts))
			}
		}
		for conn, _ := range b.neighbsSec {
			if msgInfo.sender != conn {
				io.Copy(conn, bytes.NewReader(bts))
			}
		}
		b.neighbsmu.RUnlock()
	}
}
