package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"time"

	"github.com/nxshock/torrentdb/torrent"
)

type Database struct {
	db *sql.DB
}

var (
	db *Database
)

func initDb() error {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", config.Database.User, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.DbName) // TODO: экранирование?

	log.Printf("Connecting to database %s...", dbURL)

	var err error
	db, err = newDatabase("postgres", dbURL)

	return err
}

func newDatabase(driver, address string) (*Database, error) {
	db, err := sql.Open(driver, address)
	if err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

func (database *Database) Close() error {
	return database.db.Close()
}

func (database *Database) SearchTorrentByBtih(btih []byte) (*torrent.Torrent, error) {
	var (
		title           string
		description     string
		publicationTime time.Time
		size            uint64
	)

	err := database.db.QueryRow("SELECT title, description, publication_time, size FROM info WHERE btih = $1::bytea", btih).Scan(&title, &description, &publicationTime, &size)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	t := &torrent.Torrent{Title: title, Body: template.HTML(description), PublicationTime: publicationTime, Size: size, Btih: btih}

	return t, nil
}

func (database *Database) InsertTorrent(sourceID int, topicID int, torrent *torrent.Torrent) error {
	sql := "INSERT INTO info (source_id, topic_id, title, btih, description, publication_time, size) VALUES ($1, $2, $3, $4, $5, $6, $7)"

	_, err := db.db.Exec(sql, sourceID, topicID, torrent.Title, torrent.Btih, torrent.Body, torrent.PublicationTime, torrent.Size)

	return err
}

func (database *Database) InsertTorrentWithTx(transaction *sql.Tx, sourceID int, topicID int, torrent *torrent.Torrent) error {
	sql := "INSERT INTO info (source_id, topic_id, title, btih, description, publication_time, size) VALUES ($1, $2, $3, $4, $5, $6, $7)"

	_, err := transaction.Exec(sql, sourceID, topicID, torrent.Title, torrent.Btih, torrent.Body, torrent.PublicationTime, torrent.Size)

	return err
}

func (database *Database) SearchTorrentsByTitle(query string, sortField SortField, sortDirection SortDirection) (torrents []*torrent.Torrent, err error) {
	sql := "SELECT title, btih, description, publication_time, size FROM info WHERE to_tsvector('russian', title) @@ plainto_tsquery($1::text)"

	switch sortField {
	case FieldName, FieldSize:
		sql += " ORDER BY " + string(sortField)
	case FieldTime:
		sql += " ORDER BY publication_time"
	default:
		sortField = FieldTime
		sql += " ORDER BY publication_time"
	}

	switch sortDirection {
	case SortDirectionAsc, SortDirectionDesc:
		sql += " " + string(sortDirection)
	default:
		if sortField == FieldTime {
			sql += " " + string(SortDirectionDesc)
		} else {
			sql += " " + string(SortDirectionAsc)
		}
	}

	sql += " LIMIT 100"

	rows, err := database.db.Query(sql, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			title           string
			btih            []byte
			description     string
			publicationTime time.Time
			size            uint64
		)

		err := rows.Scan(&title, &btih, &description, &publicationTime, &size)
		if err != nil {
			return nil, err
		}

		torrents = append(torrents, &torrent.Torrent{Title: title, Body: template.HTML(description), PublicationTime: publicationTime, Size: size, Btih: btih})
	}

	return torrents, nil
}

func (database *Database) GetMaxTorrentID(sourceID int) (int, error) {
	var maxTorrentID int
	err := database.db.QueryRow("SELECT COALESCE(max(topic_id), 0) FROM info WHERE source_id = $1", sourceID).Scan(&maxTorrentID)
	if err != nil {
		return 0, err
	}

	return maxTorrentID, nil
}
