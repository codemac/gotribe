package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Game struct {
	start       time.Time
	end         time.Time
	description string
	summary     string
	location    string
}

const ShittyRFC3339 = "2006-01-02 15:04:05"

func (g *Game) String() string {
	return fmt.Sprintf("%s: %s @ %s", g.start.Local().Format(ShittyRFC3339), g.summary, g.location)
}

type SortableGames []*Game

func (s SortableGames) Len() int {
	return len(s)
}

// left < right
func (s SortableGames) Less(left, right int) bool {
	return s[left].start.Before(s[right].start)
}

func (s SortableGames) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

const iCalDateFmt = "20060102T150405Z"

func remCsv(src string) string {
	return strings.Replace(src, `\,`, ",", -1)
}

func readVcal(buf []byte) *Game {
	ret := new(Game)
	sbuf := string(buf)
	lines := strings.Split(sbuf, "\n")
	for _, v := range lines {
		field := ""
		body := ""
		dfmt := iCalDateFmt
		// find field name:
		if strings.Contains(v, ";VALUE=DATE:") {
			f := strings.SplitN(v, ";", 2)
			field = f[0]

			rest := strings.Split(f[1], ":")
			body = rest[len(rest)-1]

			dfmt = "20060102"
		} else {
			f := strings.SplitN(v, ":", 2)
			if len(f) != 2 {
				continue
			}
			field = f[0]
			body = f[1]
		}

		switch field {
		case "DTSTART":
			ret.start, _ = time.Parse(dfmt, body)
		case "DTEND":
			ret.end, _ = time.Parse(dfmt, body)
		case "DESCRIPTION":
			ret.description = remCsv(body)
		case "SUMMARY":
			ret.summary = remCsv(body)
		case "LOCATION":
			ret.location = remCsv(body)
		}
	}

	return ret
}

func deferrableMain() int {
	var err error
	var f io.ReadCloser

	f, err = os.Open("schedule.ical")
	if os.IsNotExist(err) {
		resp, err := http.Get("http://mlb.am/tix/indians_schedule_full")
		if err != nil {
			return 1
		}
		f = resp.Body
	} else {
		return 1
	}
	defer f.Close()

	// split file on BEGIN:VEVENT lines

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return 1
	}

	events := bytes.Split(buf, []byte("BEGIN:VEVENT"))
	games := make([]*Game, 0)
	for _, v := range events[1:] {
		games = append(games, readVcal(v))
	}

	// find todays or next game
	sort.Sort(SortableGames(games))

	if len(os.Args) != 1 {
		for _, v := range games {
			fmt.Printf("%s\n", v)
		}
	}
	// find the first game after today at 00:00
	today := time.Now().UTC().Truncate(time.Hour * 24)
	todays_game := 0
	for k, v := range games {
		if v.start.After(today) {
			todays_game = k
			break
		}
	}

	for k, v := range games {
		d := math.Abs(float64(todays_game - k))
		switch {
		case d == 0:
			fmt.Printf("%s\n", v)
		case d < 8:
			fmt.Printf("%s\n", v)
		}
	}

	return 0
}

func main() {
	os.Exit(deferrableMain())
}
