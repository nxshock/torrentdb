package sources

import (
	"fmt"
	"sync"

	"github.com/nxshock/torrentdb/torrent"
)

var (
	sourcesMap   = make(map[string]SourceDriver)
	sourcesMutex sync.RWMutex
)

type SourceDriver interface {
	Open(string) (Source, error)
}

// Источник торрентов
type Source interface {
	// ID источника
	ID() int

	// Имя источника
	Name() string

	// Шаблон ссылки на страницу с информацией о торренте
	GetTorrentByID(int) (*torrent.Torrent, error)

	// Максимальный доступный ID торрента
	MaxTorrentID() (int, error)
}

func Register(name string, sourceDriver SourceDriver) {
	sourcesMutex.Lock()
	defer sourcesMutex.Unlock()

	if sourceDriver == nil {
		panic("sources: Register source is nil")
	}

	if _, dup := sourcesMap[name]; dup {
		panic("sql: Register called twice for source " + name)
	}
	sourcesMap[name] = sourceDriver
}

func Open(driverName, params string) (Source, error) {
	sourcesMutex.RLock()
	driveri, ok := sourcesMap[driverName]
	sourcesMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf(`unknown driver "%q" (forgotten import?)`, driverName)
	}

	return driveri.Open(params)
}

func RegisteredDrivers() []string {
	var registeredDrivers []string

	for driverName := range sourcesMap {
		registeredDrivers = append(registeredDrivers, driverName)
	}

	return registeredDrivers
}
