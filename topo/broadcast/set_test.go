package broadcast

import (
	"testing"
	"time"
)

func TestSet(t *testing.T) {
	s := NewExpiringSet(10 * time.Millisecond)
	s.Add("foo")

	<-time.After(5 * time.Millisecond)

	if !s.Has("foo") {
		t.Fatal("entry expired too early")
	}

	<-time.After(10 * time.Millisecond)

	if s.Has("foo") {
		t.Fatal("entry didn't expire")
	}
}
