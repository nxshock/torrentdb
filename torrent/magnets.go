package torrent

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type UrnType string

const (
	BitTorrent = UrnType("btih")
)

type MagnetLink struct {
	UrnType        UrnType
	UrnHash        []byte
	TrackerAddress string
}

func ParseMagnet(s string) (MagnetLink, error) {
	u, err := url.Parse(s)
	if err != nil {
		return MagnetLink{}, err
	}

	values := u.Query()

	urnType, urnHash, err := parseUrn(values.Get("xt"))
	if err != nil {
		return MagnetLink{}, err
	}

	magnetLink := MagnetLink{
		UrnType:        urnType,
		UrnHash:        urnHash,
		TrackerAddress: values.Get("tr")}

	return magnetLink, nil
}

func (m *MagnetLink) String() string {
	return fmt.Sprintf("magnet:?xt=urn:%s:%x&tr=%s", m.UrnType, m.UrnHash, m.TrackerAddress)
}

func parseUrn(s string) (urnType UrnType, urnHash []byte, err error) {
	if !strings.HasPrefix(s, "urn:") {
		return "", nil, errors.New("no urn prefix")
	}

	s = strings.TrimPrefix(s, "urn:")

	var p int
	if p = strings.Index(s, ":"); p == -1 {
		return "", nil, errors.New("?")
	}

	urnType = UrnType(s[:p])

	s = strings.TrimPrefix(s, string(urnType)+":")

	switch urnType {
	case BitTorrent:
		urnHash, err = hex.DecodeString(s)
	}
	if err != nil {
		return "", nil, err
	}

	return urnType, urnHash, nil
}

func (m *MagnetLink) HashStr() (string, error) {
	switch m.UrnType {
	case BitTorrent:
		return hex.EncodeToString(m.UrnHash), nil
	}

	return "", fmt.Errorf("unknown type: %v", m.UrnType)
}
