package rutracker

import (
	"bytes"
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

	title, err := parseTitle(bytes.NewReader(unicodeBytes))
	if err != nil {
		return nil, err
	}

	body, err := parseBody(bytes.NewReader(unicodeBytes))
	if err != nil {
		return nil, err
	}

	magnet, err := parseMagnet(bytes.NewReader(unicodeBytes))
	if err != nil {
		return nil, err
	}

	publicationTime, err := parsePublicationTime(bytes.NewReader(unicodeBytes))
	if err != nil {
		return nil, err
	}

	size, err := parseSize(bytes.NewReader(unicodeBytes))
	if err != nil {
		return nil, err
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

func parseMagnet(r io.Reader) ([]byte, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	magnetStr, exists := doc.Find("a.magnet-link").First().Attr("href")
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

	timeStr := doc.Find("div.post_head > p.post-time > span.hl-scrolled-to-wrap > a.p-link").First().Text()
	if timeStr == "" {
		return time.Time{}, errors.New("time not found")
	}

	replaceMap := map[string]string{
		"Янв": "Jan",
		"Фев": "Feb",
		"Мар": "Mar",
		"Апр": "Apr",
		"Май": "May",
		"Июн": "Jun",
		"Июл": "Jul",
		"Авг": "Aug",
		"Сен": "Sep",
		"Окт": "Oct",
		"Ноя": "Nov",
		"Дек": "Dec"}

	runes := []rune(timeStr)

	timeStr = string(runes[:3]) + replaceMap[string(runes[3:6])] + string(runes[6:])

	return time.ParseInLocation("02-Jan-06 15:04", timeStr, time.Local)
}

func parseSize(r io.Reader) (uint64, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return 0, err
	}

	sizeStr := doc.Find("fieldset.attach > div.attach_link.guest > ul.inlined.middot-separated > li").Eq(1).Text()
	if sizeStr == "" {
		return 0, errors.New("size tag not found")
	}

	fields := strings.Fields(sizeStr)
	if l := len(fields); l != 2 {
		return 0, fmt.Errorf("expeced 2 fields in size, got %d", l)
	}

	f, err := strconv.ParseFloat(fields[0], 10)
	if err != nil {
		return 0, err
	}

	var m uint64
	switch fields[1] {
	case "B":
		m = 1
	case "KB":
		m = 1024
	case "MB":
		m = 1024 * 1024
	case "GB":
		m = 1024 * 1024 * 1024
	case "TB":
		m = 1024 * 1024 * 1024 * 1024
	case "PB":
		m = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown multyplier: %s", fields[1])
	}

	return uint64(f * float64(m)), nil
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

	return 5904730, nil
}
