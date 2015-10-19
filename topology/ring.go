package topology

type Ring struct {
	me      int
	sl      []int
	peersN  int
	compare func(int, int) int
}

func NewRing(me, peersN int, compare func(int, int) int) *Ring {
	r := new(Ring)
	r.me = me
	r.peersN = peersN
	r.compare = compare
	return r
}

func (r *Ring) AddPeer(p int) (int, bool) {

	// find where to insert p
	i, same := r.closest(p)
	if same {
		return i, true
	}

	if len(r.sl) < r.peersN {
		// make room
		r.sl = append(r.sl, 0)

		// insert p at sl[i] and shift the right side by one
		copy(r.sl[i+1:], r.sl[i:len(r.sl)-1])
		r.sl[i] = p
		return 0, false
	} else {
		// closest peer to me according to compare method
		j, _ := r.closest(r.me)
		// furthest peer index-wise
		j = (j + r.peersN/2) % r.peersN

		// insert (i, p) and remove (j, rm)
		rm := r.sl[j]
		if i < j {
			copy(r.sl[i+1:j+1], r.sl[i:j])
			r.sl[i] = p
		} else {
			copy(r.sl[j:i], r.sl[j+1:i+1])
			r.sl[i-1] = p
		}
		return rm, true
	}

}

//finds closest or closest+1 element's index
func (r *Ring) closest(trg int) (int, bool) {
	for i, el := range r.sl {
		cmp := r.compare(trg, el)
		if cmp == 0 {
			return i, true
		} else if cmp < 0 {
			return i, false
		}
	}
	return len(r.sl), false
}

func (r *Ring) Peers() []int {
	return r.sl
}
