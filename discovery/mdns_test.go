package discovery

import (
	"fmt"
	"testing"
)

// Before running examples in this file, compile them with
//   go test github.com/Gaboose/go-pubsub/discovery -c
// Then follow instructions next to each example.

// Run with
//   ./discovery.test -test.run TestExamplePublish -test.v
func ExamplePublish() {
	PublishMDNS(8000)
	fmt.Println("Publishing...")
	select {}
}

// Run with
//   ./discovery.test -test.run TestExampleLookup -test.v
func ExampleLookup() {
	entries := LookupMDNS()
	fmt.Printf("Found %d entry(-ies):\n", len(entries))
	for _, ent := range entries {
		fmt.Println(ent)
	}

}

func TestExamplePublish(t *testing.T) {
	if testing.Verbose() {
		ExamplePublish()
	}
}

func TestExampleLookup(t *testing.T) {
	if testing.Verbose() {
		ExampleLookup()
	}
}
