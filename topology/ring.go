package topology

type Ring struct {
	me       int
	peersN   int
	distance func(int, int) int
	sl       []int
}

func NewRing(me, peersN int, distance func(int, int) int) *Ring {
	r := new(Ring)
	r.me = me
	r.peersN = peersN
	r.distance = distance
	return r
}

func (r *Ring) AddPeer(p int) (int, bool) {

	// find where to insert p
	i, same := r.closest(p)
	if same {
		return i, true
	}

	// make room
	r.sl = append(r.sl, 0)

	// shift the right side by one and insert (i, p)
	copy(r.sl[i+1:], r.sl[i:len(r.sl)-1])
	r.sl[i] = p

	if len(r.sl) <= r.peersN {
		return 0, false
	} else {
		// remove the furthest peer index-wise
		j, _ := r.closest(r.me)
		if j > r.peersN/2 {
			rm := r.sl[0]
			r.sl = r.sl[1:]
			return rm, true
		} else {
			rm := r.sl[r.peersN]
			r.sl = r.sl[:r.peersN]
			return rm, true
		}
	}

}

// finds where trg would go in the sorted r.sl
func (r *Ring) closest(trg int) (int, bool) {
	trgpos := r.distance(trg, r.me)
	for i, el := range r.sl {
		d := trgpos - r.distance(el, r.me)
		if d == 0 {
			return i, true
		} else if d < 0 {
			return i, false
		}
	}
	return len(r.sl), false
}

func (r *Ring) Peers() []int {
	return r.sl
}
