package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/nxshock/torrentdb/sources"
)

var errDatabaseIsUpToDate = errors.New("database is up to date")

func parserThread(transaction *sql.Tx, source sources.Source, c chan int, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()

	for id := range c {
		torrent, err := source.GetTorrentByID(id)
		if err != nil {
			errChan <- err
			continue
		}
		err = db.InsertTorrent(transaction, source.ID(), id, torrent)
		if err != nil {
			errChan <- err
			continue
		}
		errChan <- nil
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

	wg := new(sync.WaitGroup)
	wg.Add(config.Main.UpdateThreadCount)

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	errCounter := make(chan error)

	for i := 0; i < config.Main.UpdateThreadCount; i++ {
		go parserThread(tx, source, c, wg, errCounter)
	}

	go func() {
		for i := maxDbTorrentID + 1; i <= maxSourceTorrentID; i++ {
			c <- i
		}
		close(c)
		wg.Wait()
		close(errCounter)
	}()

	var (
		newTorrentsCount int
		errorCount       int
	)
	for err := range errCounter {
		if err == nil {
			errorCount++
		} else {
			newTorrentsCount++
		}
		fmt.Fprintf(os.Stderr, "\rProcessed %d / %d (new torrents: %d, errors: %d)...", errorCount+newTorrentsCount, maxSourceTorrentID-maxDbTorrentID, newTorrentsCount, errorCount)
	}
	fmt.Fprintf(os.Stderr, "\n")

	err = tx.Commit()
	if err != nil {
		return err
	}

	log.Printf("Update of %s completed.", driverName)

	return nil
}
