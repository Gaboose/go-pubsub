package gway

// PeerInfo is a small struct used to pass around between
// topo subpackages. It implements topo.Peer interface.
type PeerInfo struct {
	ID     string
	MAddrs [][]byte
	Params map[string]interface{}
}

func (p PeerInfo) Id() interface{}          { return p.ID }
func (p PeerInfo) Get(k string) interface{} { return p.Params[k] }

func (p *PeerInfo) Put(k string, v interface{}) {
	if p.Params == nil {
		p.Params = make(map[string]interface{})
	}
	p.Params[k] = v
}
