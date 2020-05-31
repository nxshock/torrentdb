package rutor

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/nxshock/torrentdb/torrent"
	"golang.org/x/net/proxy"
)

const maxIDUrl = "http://new-rutor.org"
const urlMask = "http://new-rutor.org/torrent/%d"

type Parser struct {
	httpClient *http.Client
}

func newParser(socksProxyAddress string) (*Parser, error) {
	dialer, err := proxy.SOCKS5("tcp", socksProxyAddress, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	parser := &Parser{httpClient: &http.Client{Transport: &http.Transport{Dial: dialer.Dial}}}

	return parser, nil
}

func (parser *Parser) ID() int {
	return 2
}

func (parser *Parser) MaxTorrentID() (int, error) {
	resp, err := parser.httpClient.Get(maxIDUrl)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	var maxID int
	doc.Find("div#index > table > tbody > tr > td:nth-child(2) > a").Each(
		func(i int, s *goquery.Selection) {
			id := getTorrentIDFromURL(s.AttrOr("href", ""))
			if id > maxID {
				maxID = id
			}
		})

	return maxID, nil
}

func (parser *Parser) Name() string {
	return "Rutor"
}

func (parser *Parser) GetTorrentByID(id int) (*torrent.Torrent, error) {
	resp, err := parser.httpClient.Get(fmt.Sprintf(urlMask, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	title, err := parseTitle(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%d: %v", id, err)
	}

	body, err := parseBody(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%d: %v", id, err)
	}

	magnet, err := parseMagnet(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%d: %v", id, err)
	}

	publicationTime, err := parsePublicationTime(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%d: %v", id, err)
	}

	size, err := parseSize(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%d: %v", id, err)
	}

	torrent := &torrent.Torrent{
		Title:           title,
		Body:            template.HTML(body),
		Btih:            magnet,
		PublicationTime: publicationTime,
		Size:            size}

	return torrent, nil
}

func parseTitle(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	title := doc.Find("html > head > title").First().Text()
	title = strings.TrimPrefix(title, "new-rutor.org :: ")

	return title, nil
}

func parseBody(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	//body := doc.Find("table#details > tbody > tr:nth-child(1) > td:nth-child(2)").First().Text()

	/*body, err := doc.Find("table#details > tbody > tr:nth-child(1) > td:nth-child(2)").First().Html()
	if err != nil {
		return "", err
	}
	body = strings.TrimSpace(body)*/

	markdown := md.NewConverter("", true, nil).Convert(doc.Find("table#details > tbody > tr:nth-child(1) > td:nth-child(2)").First())
	return markdown, nil
	//return body, nil
}

func parseMagnet(r io.Reader) ([]byte, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	magnetStr, exists := doc.Find("div#download > a").First().Attr("href")
	if !exists {
		return nil, errors.New("href attr does not exists")
	}

	magnet, err := torrent.ParseMagnet(magnetStr)
	if err != nil {
		return nil, err
	}

	return magnet.UrnHash, nil
}

func parsePublicationTime(r io.Reader) (time.Time, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return time.Time{}, err
	}

	var timeStr string
	doc.Find("table#details > tbody > tr").EachWithBreak(
		func(i int, s *goquery.Selection) bool {
			if s.Find("td:nth-child(1)").Text() == "Добавлен" {
				timeStr = s.Find("td:nth-child(2)").Text()
				return false
			}

			return true
		})
	if timeStr == "" {
		return time.Time{}, errors.New("time not found")
	}

	p := strings.Index(timeStr, "  ")
	if p < 0 {
		return time.Time{}, fmt.Errorf("unexpected time format: %s", timeStr)
	}

	return time.ParseInLocation("02-01-2006 15:04:05", timeStr[:p], time.Local)
}

func parseSize(r io.Reader) (uint64, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return 0, err
	}

	var sizeStr string
	doc.Find("table#details > tbody > tr").EachWithBreak(
		func(i int, s *goquery.Selection) bool {
			if s.Find("td:nth-child(1)").Text() == "Размер" {
				sizeStr = s.Find("td:nth-child(2)").Text()
				return false
			}

			return true
		})
	if sizeStr == "" {
		return 0, errors.New("size tag not found")
	}

	fields := strings.Fields(sizeStr)
	if l := len(fields); l != 4 {
		return 0, fmt.Errorf("expeced 4 fields in size, got %d", l)
	}

	f, err := strconv.ParseUint(fields[2][1:], 10, 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}

func getTorrentIDFromURL(s string) int {
	s = strings.TrimPrefix(s, "/torrent/")
	if p := strings.Index(s, "/"); p < 0 {
		return 0
	} else {
		s = s[:p]
	}

	id, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return id
}
