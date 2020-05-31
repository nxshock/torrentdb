package torrent

import (
	"encoding/hex"
	"fmt"
	"html/template"
	"time"
)

type Torrent struct {
	Title           string
	Body            template.HTML
	Btih            []byte
	PublicationTime time.Time
	Size            uint64
}

func (t *Torrent) HumanSize() template.HTML {
	type sizeInfo struct {
		s    uint64
		name string
	}

	sizesInfo := []sizeInfo{
		{1 << 50, "PiB"},
		{1 << 40, "TiB"},
		{1 << 30, "GiB"},
		{1 << 20, "MiB"},
		{1 << 10, "KiB"}}

	for _, v := range sizesInfo {
		if t.Size/v.s > 0 {
			return template.HTML(fmt.Sprintf("%.1f&nbsp;%s", float64(t.Size)/float64(v.s), v.name))
		}
	}

	return template.HTML(fmt.Sprintf("%d&nbsp;%s", t.Size, "B"))
}

func (t *Torrent) HumanTime() template.HTML {
	return template.HTML(t.PublicationTime.Format("02.01.06&nbsp;15:04"))
}

func (t *Torrent) BtihHex() string {
	return hex.EncodeToString(t.Btih)
}
