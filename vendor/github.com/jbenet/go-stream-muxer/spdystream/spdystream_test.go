package peerstream_spdystream

import (
	"testing"

	test "github.com/jbenet/go-stream-muxer/test"
)

func TestSpdyStreamTransport(t *testing.T) {
	test.SubtestAll(t, Transport)
}
