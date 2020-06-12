package main

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/nxshock/torrentdb/sources"
)

var errDatabaseIsUpToDate = errors.New("database is up to date")

func parserThread(source sources.Source, c chan int, wg *sync.WaitGroup, errorCount *int64) {
	defer wg.Done()

	for id := range c {
		torrent, err := source.GetTorrentByID(id)
		if err != nil {
			atomic.AddInt64(errorCount, 1)
			continue
		}
		err = db.InsertTorrent(source.ID(), id, torrent)
		if err != nil {
			log.Println(err)
		}
	}
}

func updateAll() {
	drivers := sources.RegisteredDrivers()
	for _, driverName := range drivers {
		err := update(driverName)
		if err == errDatabaseIsUpToDate {
			log.Printf("%s: %v", driverName, err)
		} else if err != nil {
			log.Printf("Update %s torrent data error: %v", driverName, err)
		}
	}
}

func update(driverName string) error {
	source, err := sources.Open(driverName, config.Main.ProxyAddr)
	if err != nil {
		return err
	}

	maxDbTorrentID, err := db.GetMaxTorrentID(source.ID())
	if err != nil {
		return err
	}
	maxSourceTorrentID, err := source.MaxTorrentID()
	if err != nil {
		return err
	}

	if maxDbTorrentID >= maxSourceTorrentID {
		return errDatabaseIsUpToDate
	}

	log.Printf("Обновление данных %s с %d по %d", driverName, maxDbTorrentID+1, maxSourceTorrentID)

	c := make(chan int)

	var errorCount int64

	wg := new(sync.WaitGroup)
	wg.Add(config.Main.UpdateThreadCount)
	for i := 0; i < config.Main.UpdateThreadCount; i++ {
		go parserThread(source, c, wg, &errorCount)
	}

	for i := maxDbTorrentID + 1; i <= maxSourceTorrentID; i++ {
		c <- i
	}

	close(c)

	wg.Wait()

	log.Printf("Update of %s completed. New torrents: %d, errors: %d.", driverName, (maxSourceTorrentID - maxDbTorrentID - 1 - int(errorCount)), errorCount)

	return nil
}
