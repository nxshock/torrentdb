package rutor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParserMaxTorrentID(t *testing.T) {
	parser, err := newParser("192.168.0.2:9050")
	assert.NoError(t, err)

	maxID, err := parser.MaxTorrentID()
	assert.NoError(t, err)

	assert.True(t, maxID > 0)
}

func TestGetTorrentIDFromURL(t *testing.T) {
	tests := []struct {
		s          string
		expectedID int
	}{
		{"/torrent/758938/hroniki-narnii-princ-kaspian_the-chronicles-of-narnia-prince-caspian-2008-web-dlrip-720p-ot-supermin-d-open-matte", 758938},
		{"/torrent/758917/udivitelnoe-puteshestvie-doktora-dulittla_dolittle-2020-uhd-bdremux-2160p-ot-selezen-4k-hdr-d-p-licenzija", 758917},
	}

	for _, test := range tests {
		gotID := getTorrentIDFromURL(test.s)
		assert.Equal(t, test.expectedID, gotID)
	}
}
