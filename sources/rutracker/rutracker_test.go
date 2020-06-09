package rutracker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTorrentByIDFromApi(t *testing.T) {
	parser, err := newParser("192.168.0.2:9050")
	assert.NoError(t, err)

	torrent, err := parser.getTorrentInfoFromApi(5896188)
	assert.NoError(t, err)

	assert.Equal(t, time.Unix(1591711281, 0), torrent.PublicationTime)
	assert.Equal(t, "55fcd06474e50f49003f7e93681763afaa4d506d", torrent.BtihHex())
	assert.Equal(t, uint64(524722552), torrent.Size)
	assert.Equal(t, "[Nintendo Switch] Dungeon of the Endless + Deep Freeze, Death Gamble, Rescue Team, Organic Matters [NSZ][ENG]", torrent.Title)
}
