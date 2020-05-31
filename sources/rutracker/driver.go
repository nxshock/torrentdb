package rutracker

import (
	"github.com/nxshock/torrentdb/sources"
)

type driver struct{}

var driverInterface = new(driver)

func (driver *driver) Open(proxyUrl string) (sources.Source, error) {
	return newParser(proxyUrl)
}

func init() {
	sources.Register("rutracker", driverInterface)
}
