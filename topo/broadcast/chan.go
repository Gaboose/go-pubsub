package broadcast

import "fmt"

// overflowBuffer forms a last-in last-out queue between the given channels.
// Input channel never blocks from outside. If the buffer is full,
// it'll discard the last-in value.
func overflowBuffer(n int, in <-chan string, out chan<- string) {
	var outMaybe chan<- string
	//buf, i and j together form a circular buffer
	i, j := 0, 0
	buf := make([]string, n)
	for {
		select {
		case v, ok := <-in:
			if !ok {
				close(out)
				return
			}
			if i == j {
				if outMaybe == nil {
					// buffer is empty
					outMaybe = out
				} else {
					// buffer is full
					// overwrite last value and nudge the output index
					j = (j + 1) % n
				}
			}
			buf[i] = v
			i = (i + 1) % n

		case outMaybe <- buf[j]:
			j = (j + 1) % n
			if i == j {
				outMaybe = nil
			}
		}
	}
}

// gate controls the flow between 'in' and 'out' channels depending on the last
// value sent to 'ctrl'. If it was true, the gate is open, if it was false,
// the gate is closed and outside senders to channel 'in' will block.
func gate(ctrl <-chan bool, in <-chan string, out chan<- string) {
	var inMaybe <-chan string
	for {
		select {
		case b, ok := <-ctrl:
			fmt.Printf("ctrl %v\n", b)
			if !ok {
				return
			} else if b {
				inMaybe = in
			} else {
				inMaybe = nil
			}
		case v := <-inMaybe:
			out <- v
		}
	}
}
