package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"

	_ "github.com/lib/pq"

	_ "github.com/nxshock/torrentdb/sources/rutor"
	_ "github.com/nxshock/torrentdb/sources/rutracker"
)

func init() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if len(os.Args) > 2 {
		initConfig(os.Args[2])
	} else {
		initConfig(defaultConfigPath)
	}

	err := initDb()
	if err != nil {
		log.Fatalln("Connect database error: %v", err)
	}

	switch os.Args[1] {
	case "daemon":
		initServer()
	}
}

func main() {
	switch os.Args[1] {
	case "daemon":
		wait()
	case "update-all":
		updateAll()
	default:
		log.Fatalf("Unknown command: %s", os.Args[1])
	}

	db.Close()
}

func printUsage() {
	binName := filepath.Base(os.Args[0])

	log.Println("Usage:")
	log.Printf("%s daemon     [config file path] - start http server", binName)
	log.Printf("%s update-all [config file path] - update database data", binName)
}

func wait() { // TODO: нужно имя получше
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	log.Printf("Signal %v received.", <-c)
}
