package rutracker

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParserMaxTorrentID(t *testing.T) {
	parser, err := newParser("192.168.0.2:9050")
	assert.NoError(t, err)

	log.Println(parser.MaxTorrentID())
}
