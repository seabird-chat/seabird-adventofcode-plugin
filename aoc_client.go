package adventofcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"time"
)

// This is effectively time.DateTime plus the timezone name
var dateFormat = "2006-01-02 15:04:05 MST"

type Leaderboard struct {
	Event         string `json:"event"`
	NumDays       int    `json:"num_days"`
	Day1Timestamp int64  `json:"day1_ts"`
	OwnerID       int    `json:"owner_id"`

	Members map[string]Member `json:"members"`
}

func (l *Leaderboard) GetEvents() []*Event {
	var ret []*Event

	for _, member := range l.Members {
		events := member.getEvents()
		ret = append(ret, events...)
	}

	for _, event := range ret {
		event.Leaderboard = l
	}

	slices.SortFunc(ret, func(a, b *Event) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	return ret
}

type Member struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Stars             int    `json:"stars"`
	LocalScore        int    `json:"local_score"`
	LastStarTimestamp int    `json:"last_star_ts"`

	CompletionDayLevel map[string]SingleDay `json:"completion_day_level"`
}

func (m *Member) getEvents() []*Event {
	var ret []*Event

	for day, dayData := range m.CompletionDayLevel {
		if dayData.Star1 != nil {
			ret = append(ret, &Event{
				Member:    m,
				Day:       day,
				Star:      1,
				Timestamp: time.Unix(int64(dayData.Star1.Timestamp), 0).In(tzEastern),
			})
		}
		if dayData.Star2 != nil {
			ret = append(ret, &Event{
				Member:    m,
				Day:       day,
				Star:      2,
				Timestamp: time.Unix(int64(dayData.Star2.Timestamp), 0).In(tzEastern),
			})
		}
	}

	return ret
}

type SingleDay struct {
	Star1 *SingleStar `json:"1"`
	Star2 *SingleStar `json:"2"`
}

type SingleStar struct {
	Index     int `json:"star_index"`
	Timestamp int `json:"get_star_ts"`
}

type Event struct {
	Leaderboard *Leaderboard
	Member      *Member
	Day         string
	Star        int
	Timestamp   time.Time
}

func (e *Event) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "AoC %s: ", e.Leaderboard.Event)
	fmt.Fprintf(&buf, "%s completed ", e.Member.Name)
	fmt.Fprintf(&buf, "the %s star of day %s ", Ordinal(e.Star), e.Day)
	fmt.Fprintf(&buf, "at %s (%d)", e.Timestamp.Format(dateFormat), e.Timestamp.Unix())

	return buf.String()
}

type AOCClient struct {
	client *http.Client
}

func NewAOCClient(token string) (*AOCClient, error) {
	cj, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	cj.SetCookies(
		&url.URL{Scheme: "https", Host: "adventofcode.com"},
		[]*http.Cookie{{Name: "session", Value: token}})
	client := &http.Client{
		Jar: cj,
	}
	return &AOCClient{
		client: client,
	}, nil
}

func (c *AOCClient) GetLeaderboard(ctx context.Context, event string, id string) (*Leaderboard, error) {
	leaderboardUrl := fmt.Sprintf("https://adventofcode.com/%s/leaderboard/private/view/%s.json", event, id)
	req, err := http.NewRequest("GET", leaderboardUrl, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	req.Header.Add("User-Agent", "seabird-adventofcode-plugin (kaleb@coded.io)")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var leaderboard Leaderboard
	err = json.Unmarshal(data, &leaderboard)
	if err != nil {
		return nil, err
	}

	return &leaderboard, nil
}
