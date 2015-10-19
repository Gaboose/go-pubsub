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
	r.AddPeer(11)
	r.AddPeer(12)
	r.AddPeer(3)
	r.AddPeer(1)

	got := r.Peers()
	exp := []int{1, 3, 10, 11}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}

	r.AddPeer(2)
	got = r.Peers()
	exp = []int{2, 3, 10, 11}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}
}
