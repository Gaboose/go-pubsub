package main

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
)

func TestWriteReadClose(t *testing.T) {
	c1, c2 := net.Pipe()

	var s1, s2 ErrorCodeRWC
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		s1, _ = WithErrorCode(c1)
		wg.Done()
	}()
	go func() {
		s2, _ = WithErrorCode(c2)
		wg.Done()
	}()
	wg.Wait()

	toWrite := []byte("hello")
	_, err := s1.Write(toWrite)
	if err != nil {
		t.Fatal(err)
	}

	s1.Close(5)

	var toRead bytes.Buffer
	_, err = io.Copy(&toRead, s2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(toWrite, toRead.Bytes()) {
		t.Fatalf("written %v, but read %v", toWrite, toRead)
	}

	if ec := <-s2.ErrorCodeCh(); ec != 5 {
		t.Fatalf("closed with 5, but got %d", ec)
	}
}
