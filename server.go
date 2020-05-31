package main

import (
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"gopkg.in/russross/blackfriday.v2"

	"github.com/nxshock/torrentdb/torrent"
)

var templates *template.Template

func initServer() {
	templatesDir := filepath.Join(config.Main.SiteDir)

	log.Printf("Чтение списка шаблонов из %s...", templatesDir)
	files, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Чтение списка шаблонов завершено.")

	templates, err = template.New("").ParseFiles(files...)
	if err != nil {
		log.Fatalln(">", err)
	}

	http.HandleFunc("/torrent", torrentHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/", rootHandler)

	go func() {
		err := http.ListenAndServe(":80", nil)
		if err != nil {
			log.Fatalln(err)
		}
	}()
}

func torrentHandler(w http.ResponseWriter, r *http.Request) {
	btih, err := hex.DecodeString(r.FormValue("btih"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	torrent, err := db.SearchTorrentByBtih(btih)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Подготовка markdown
	torrent.Body = template.HTML(strings.ReplaceAll(string(torrent.Body), "\n\n", "<newline>"))
	torrent.Body = template.HTML(strings.ReplaceAll(string(torrent.Body), "\n", "<br>"))
	torrent.Body = template.HTML(strings.ReplaceAll(string(torrent.Body), "<newline>", "\n\n"))
	torrent.Body = template.HTML(string(blackfriday.Run([]byte(torrent.Body))))

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	templates.ExecuteTemplate(w, "torrent.html", torrent)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	type TemplateData = struct {
		Query          string
		OrderBy        SortField
		OrderDirection SortDirection
		List           []*torrent.Torrent
	}

	query := r.FormValue("query") // TODO: отфильтровать запрос
	sortBy := SortField(r.FormValue("orderBy"))
	sortDirection := SortDirection(r.FormValue("orderDirection"))

	templateData := TemplateData{
		Query:          query,
		OrderBy:        sortBy,
		OrderDirection: sortDirection}

	if query == "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		templates.ExecuteTemplate(w, "search.html", templateData)
		return
	}

	torrents, err := db.SearchTorrentsByTitle(query, sortBy, sortDirection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templateData.List = torrents

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	templates.ExecuteTemplate(w, "search.html", templateData)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	switch r.RequestURI[1:] {
	case "", "index.html", "index.htm":
		indexHandler(w, r)
	default:
		http.ServeFile(w, r, filepath.Join(config.Main.SiteDir, r.RequestURI[1:]))
	}

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	templates.ExecuteTemplate(w, "index.html", nil)
}
