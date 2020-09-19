package rutracker

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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
	"golang.org/x/net/proxy"
	"golang.org/x/text/encoding/charmap"

	"github.com/nxshock/torrentdb/torrent"
)

const urlMask = "https://rutracker.org/forum/viewtopic.php?t=%d"

type Parser struct {
	httpClient *http.Client
}

func newParser(socksProxyAddress string) (*Parser, error) {
	httpTransport := http.DefaultTransport

	if socksProxyAddress != "" {
		dialer, err := proxy.SOCKS5("tcp", socksProxyAddress, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}

		httpTransport = &http.Transport{Dial: dialer.Dial}
	}

	parser := &Parser{httpClient: &http.Client{Transport: httpTransport}}

	return parser, nil
}

func (parser *Parser) ID() int {
	return 1
}

func (parser *Parser) Name() string {
	return "RuTracker"
}

func (parser *Parser) GetTorrentByID(id int) (*torrent.Torrent, error) {
	torrent, err := parser.getTorrentInfoFromApi(id)
	if err != nil {
		return nil, err
	}

	resp, err := parser.httpClient.Get(fmt.Sprintf(urlMask, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	win1251Bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	unicodeBytes, err := charmap.Windows1251.NewDecoder().Bytes(win1251Bytes)
	if err != nil {
		return nil, err
	}

	body, err := parseBody(bytes.NewReader(unicodeBytes))
	if err != nil {
		return nil, err
	}

	torrent.Body = template.HTML(body)

	return torrent, nil
}

func parseTitle(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	title := doc.Find("html > head > title").First().Text()
	title = strings.TrimSuffix(title, " :: RuTracker.org")

	return title, nil
}

func parseBody(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	// Удаление лишних заголовков
	doc.Find("div.c-head").Remove()

	for _, node := range doc.Find("div.sp-head").Nodes {
		node.Data = "h3"
	}

	// TODO: удаление лишних переносов строк
	doc.Find("div.c-body").Each(func(n int, s *goquery.Selection) {
		s.SetText(strings.TrimSpace(s.Text()))
	})

	// Преобразование блоков с кодом
	for _, node := range doc.Find("div.c-body").Nodes {
		node.Data = "pre"
	}

	// Починка картинок
	doc.Find("var.postImg").Each(func(i int, s *goquery.Selection) {
		s.SetHtml(fmt.Sprintf(`<img src="%s">`, s.AttrOr("title", "")))
	})

	//html, _ := doc.Find("table#topic_main div.post_body").First().Html()
	//ioutil.WriteFile("123.html", []byte(html), 0644)

	markdown := md.NewConverter("", true, nil).Convert(doc.Find("table#topic_main div.post_body").First())
	return markdown, nil
}

// TODO: без регистрации на сайте не работает, требуется альтернативный алгоритм
func (parser *Parser) MaxTorrentID() (int, error) {
	type Resp struct {
		Result map[int][]int
	}

	const url = "http://api.rutracker.org/v1/static/forum_size"

	resp, err := parser.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var jResp Resp
	err = json.NewDecoder(resp.Body).Decode(&jResp)
	if err != nil {
		return 0, err
	}

	var maxID int
	for _, group := range jResp.Result {
		maxID += group[0]
	}

	return 5943785, nil
}

func (parser *Parser) getTorrentInfoFromApi(id int) (*torrent.Torrent, error) {
	type ApiResp struct {
		Result map[int]*struct {
			InfoHash   string  `json:"info_hash"`
			ForumID    int     `json:"forum_id"`
			Size       float64 `json:"size"`
			RegTime    int64   `json:"reg_time"`
			TopicTitle string  `json:"topic_title"`
		}
	}

	resp, err := parser.httpClient.Get("http://api.rutracker.org/v1/get_tor_topic_data?by=topic_id&val=" + strconv.Itoa(id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp ApiResp
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, err
	}

	info := apiResp.Result[id]
	if info == nil {
		return nil, errors.New("no such torrent")
	}

	urnHash, err := hex.DecodeString(info.InfoHash)
	if info == nil {
		return nil, fmt.Errorf("wrong urn hash: %v", err)
	}

	t := &torrent.Torrent{
		Title:           info.TopicTitle,
		Size:            uint64(info.Size),
		PublicationTime: time.Unix(info.RegTime, 0),
		Btih:            urnHash}

	return t, nil
}
