package broadcast

import "sort"

type OverflowSlice []Peer

// Insert adds a new peer to the slice. If the slice is overflowing,
// Insert removes the oldest peer. The slice is kept sorted in any case.
func (s OverflowSlice) Insert(p Peer) OverflowSlice {
	bd := p.GetBday()
	i := sort.Search(len(s), func(i int) bool {
		return s[i].GetBday() <= bd
	})

	if i < cap(s) {
		if len(s) < cap(s) {
			s = s[:len(s)+1]
		}
		copy(s[i+1:], s[i:])
		s[i] = p
	}

	return s
}

// UntilFirst calls f for each element until f returns true.
// UntilFirst returns a slice containing the remaining peers.
func (s OverflowSlice) UntilFirst(f func(Peer) bool) OverflowSlice {
	tail := s
	for len(tail) > 0 {
		p := tail[0]
		tail = tail[1:]
		if f(p) {
			break
		}
	}
	return tail
}
