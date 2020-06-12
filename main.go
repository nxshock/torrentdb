package main

import (
	"fmt"
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

	// TODO: init config from specified param?
	initConfig(defaultConfigPath)

	err := initDb()
	if err != nil {
		log.Fatalln("Connect database error: %v", err)
	}

	switch os.Args[1] {
	case "daemon":
		initServer()
	}

	switch os.Args[1] {
	case "update":
		if len(os.Args) != 3 {
			printUsage()
			os.Exit(1)
		}
	}
}

func main() {
	var err error

	switch os.Args[1] {
	case "daemon":
		wait()
	case "update-all":
		updateAll()
	case "update":
		err = update(os.Args[2])
	default:
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}

	db.Close()

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

}

func printUsage() {
	binName := filepath.Base(os.Args[0])

	log.Println("Usage:")
	log.Printf("%s daemon               - start http server", binName)
	log.Printf("%s update [source_name] - update specified database data", binName)
	log.Printf("%s update-all           - update database data", binName)
}

func wait() { // TODO: нужно имя получше
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	log.Printf("Signal %v received.", <-c)
}
