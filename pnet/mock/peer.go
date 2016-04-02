package mock

import "fmt"

type Peer struct {
	ID     string
	Params map[string]interface{}
}

func (p Peer) Id() interface{}          { return p.ID }
func (p Peer) Get(k string) interface{} { return p.Params[k] }
func (p *Peer) Put(k string, v interface{}) {
	if p.Params == nil {
		p.Params = make(map[string]interface{})
	}
	p.Params[k] = v
}

func (p Peer) String() string {
	return fmt.Sprintf("{%s %v}", p.ID, p.Params)
}
