package peer

import "time"
import ma "github.com/jbenet/go-multiaddr"

type expiringAddr struct {
	Addr ma.Multiaddr
	TTL  time.Time
}

func (e *expiringAddr) ExpiredBy(t time.Time) bool {
	return t.After(e.TTL)
}

type addrSet map[string]expiringAddr

type AddrManager struct {
	addrs map[ID]addrSet
}

// ensures the AddrManager is initialized.
// So we can use the zero value.
func (mgr *AddrManager) init() {
	if mgr.addrs == nil {
		mgr.addrs = make(map[ID]addrSet)
	}
}

func (mgr *AddrManager) Peers() []ID {
	if mgr.addrs == nil {
		return nil
	}

	pids := make([]ID, 0, len(mgr.addrs))
	for pid := range mgr.addrs {
		pids = append(pids, pid)
	}
	return pids
}

// AddAddr calls AddAddrs(p, []ma.Multiaddr{addr}, ttl)
func (mgr *AddrManager) AddAddr(p ID, addr ma.Multiaddr, ttl time.Duration) {
	mgr.AddAddrs(p, []ma.Multiaddr{addr}, ttl)
}

// AddAddrs gives AddrManager addresses to use, with a given ttl
// (time-to-live), after which the address is no longer valid.
// If the manager has a longer TTL, the operation is a no-op for that address
func (mgr *AddrManager) AddAddrs(p ID, addrs []ma.Multiaddr, ttl time.Duration) {
	// if ttl is zero, exit. nothing to do.
	if ttl <= 0 {
		return
	}

	// so zero value can be used
	mgr.init()

	amap, found := mgr.addrs[p]
	if !found {
		amap = make(addrSet)
		mgr.addrs[p] = amap
	}

	// only expand ttls
	exp := time.Now().Add(ttl)
	for _, addr := range addrs {
		addrstr := addr.String()
		a, found := amap[addrstr]
		if !found || exp.After(a.TTL) {
			amap[addrstr] = expiringAddr{Addr: addr, TTL: exp}
		}
	}
}

// SetAddr calls mgr.SetAddrs(p, addr, ttl)
func (mgr *AddrManager) SetAddr(p ID, addr ma.Multiaddr, ttl time.Duration) {
	mgr.SetAddrs(p, []ma.Multiaddr{addr}, ttl)
}

// SetAddrs sets the ttl on addresses. This clears any TTL there previously.
// This is used when we receive the best estimate of the validity of an address.
func (mgr *AddrManager) SetAddrs(p ID, addrs []ma.Multiaddr, ttl time.Duration) {
	// so zero value can be used
	mgr.init()

	amap, found := mgr.addrs[p]
	if !found {
		amap = make(addrSet)
		mgr.addrs[p] = amap
	}

	exp := time.Now().Add(ttl)
	for _, addr := range addrs {
		// re-set all of them for new ttl.
		addrs := addr.String()

		if ttl > 0 {
			amap[addrs] = expiringAddr{Addr: addr, TTL: exp}
		} else {
			delete(amap, addrs)
		}
	}
}

// Addresses returns all known (and valid) addresses for a given
func (mgr *AddrManager) Addrs(p ID) []ma.Multiaddr {
	// not initialized? nothing to give.
	if mgr.addrs == nil {
		return nil
	}

	maddrs, found := mgr.addrs[p]
	if !found {
		return nil
	}

	now := time.Now()
	good := make([]ma.Multiaddr, 0, len(maddrs))
	var expired []string
	for s, m := range maddrs {
		if m.ExpiredBy(now) {
			expired = append(expired, s)
		} else {
			good = append(good, m.Addr)
		}
	}

	// clean up the expired ones.
	for _, s := range expired {
		delete(maddrs, s)
	}
	return good
}
