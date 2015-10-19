package topology

import (
	"reflect"
	"testing"
)

func TestSingle(t *testing.T) {
	r := NewRing(5, 2, func(a, b int) int { return a - b })
	r.AddPeer(10)
	got := r.Peers()
	if !reflect.DeepEqual(got, []int{10}) {
		t.Error("Expected [10], got", got)
	}
}

func TestMultiple(t *testing.T) {
	r := NewRing(5, 4, func(a, b int) int { return a - b })
	r.AddPeer(10)
	r.AddPeer(2)
	r.AddPeer(11)
	r.AddPeer(0)
	r.AddPeer(12)

	got := r.Peers()
	exp := []int{0, 2, 10, 11}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}

	r.AddPeer(3)
	got = r.Peers()
	exp = []int{2, 3, 10, 11}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}
}

func TestWrap(t *testing.T) {
	r := NewRing(20, 4, modDist(100))
	r.AddPeer(10)
	r.AddPeer(30)
	r.AddPeer(60)
	r.AddPeer(80)

	got := r.Peers()
	exp := []int{80, 10, 30, 60}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}
}

func modDist(mod int) func(int, int) int {
	hmod := mod / 2
	return func(a, b int) int {
		return (a-b+hmod)%mod - hmod
	}
}
