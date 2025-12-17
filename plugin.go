package adventofcode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/seabird-chat/seabird-go"
	"github.com/seabird-chat/seabird-go/pb"
	"golang.org/x/sync/errgroup"
)

func init() {
	eastern, err := time.LoadLocation("US/Eastern")
	if err != nil {
		panic(err)
	}
	tzEastern = eastern
}

var (
	tzEastern                *time.Location
	leaderboardTickFrequency = 15 * time.Minute
	topScoreCount            = 5
)

type Config struct {
	SeabirdHost    string
	SeabirdToken   string
	AOCSession     string
	AOCLeaderboard string
	AOCChannel     string
	TimestampFile  string
}

type Plugin struct {
	logger      *slog.Logger
	config      Config
	sbClient    *seabird.Client
	aocClient   *AOCClient
	queueUpdate chan chan bool

	cacheLock         sync.RWMutex
	cachedLeaderboard *Leaderboard
}

func NewPlugin(logger *slog.Logger, config Config) (*Plugin, error) {
	sbClient, err := seabird.NewClient(config.SeabirdHost, config.SeabirdToken)
	if err != nil {
		return nil, err
	}

	aocClient, err := NewAOCClient(config.AOCSession)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		logger:      logger,
		config:      config,
		sbClient:    sbClient,
		aocClient:   aocClient,
		queueUpdate: make(chan chan bool),
	}, nil
}

func (p *Plugin) runSeabirdStream(ctx context.Context) error {
	events, err := p.sbClient.StreamEvents(map[string]*pb.CommandMetadata{
		"aoc-status": {
			Name:      "aoc",
			ShortHelp: "[year]",
			FullHelp:  "Look up the AOC leaderboard for a given year",
		},
		"aoc-refresh": {
			Name:      "aoc-refresh",
			ShortHelp: "",
			FullHelp:  "Refresh the Advent of Code leaderboard",
		},
	})
	if err != nil {
		return err
	}
	defer events.Close()

	for event := range events.C {
		switch v := event.GetInner().(type) {
		case *pb.Event_Command:
			switch v.Command.GetCommand() {
			case "aoc-refresh":
				go p.handleRefresh(ctx, v.Command)
			case "aoc-status", "aoc", "advent":
				go p.handleStatus(ctx, v.Command)
			}
		}
	}

	return errors.New("event stream unexpectedly closed")
}

func (p *Plugin) handleStatus(ctx context.Context, event *pb.CommandEvent) {
	arg := strings.TrimSpace(event.Arg)

	var leaderboard *Leaderboard
	var err error

	// When no args provided, try to use cached leaderboard
	if arg == "" {
		p.cacheLock.RLock()
		leaderboard = p.cachedLeaderboard
		p.cacheLock.RUnlock()
	}

	// If no cache or args were provided, fetch from API
	if leaderboard == nil {
		p.logger.With(slog.String("event", arg)).Info("Leaderboard not cached, calling API")
		leaderboard, err = p.lookupLeaderboard(ctx, arg)
		if err != nil {
			p.logger.With(slog.Any("error", err)).Error("Failed to lookup leaderboard")
			_ = p.sbClient.MentionReply(event.Source, "Failed to lookup leaderboard")
			return
		}
	}

	// Convert members map to slice, filtering out members with no stars
	members := make([]Member, 0, len(leaderboard.Members))
	for _, member := range leaderboard.Members {
		if member.Stars == 0 {
			continue
		}
		members = append(members, member)
	}

	// Sort by Stars descending
	slices.SortFunc(members, func(a, b Member) int {
		return b.Stars - a.Stars
	})

	// Take top N members
	if len(members) > topScoreCount {
		members = members[:topScoreCount]
	}

	// Format star counts
	var scores []string
	for _, member := range members {
		scores = append(scores, fmt.Sprintf("%s (%d)", member.Name, member.Stars))
	}

	// Send reply with event name
	reply := fmt.Sprintf("AoC %s: %s", leaderboard.Event, strings.Join(scores, ", "))
	_ = p.sbClient.Reply(event.Source, reply)
}

func (p *Plugin) handleRefresh(ctx context.Context, event *pb.CommandEvent) {
	updateResp := make(chan bool, 1)

	select {
	case p.queueUpdate <- updateResp:
		_ = p.sbClient.MentionReply(event.Source, "Queued leaderboard refresh")

		select {
		case ok := <-updateResp:
			if ok {
				_ = p.sbClient.MentionReply(event.Source, "Leaderboard successfully refreshed")
			} else {
				_ = p.sbClient.MentionReply(event.Source, "Failed to refresh leaderboard")
			}
		case <-time.After(5 * time.Second):
			_ = p.sbClient.MentionReply(event.Source, "Leaderboard refresh timed out")
		}
	default:
		_ = p.sbClient.MentionReply(event.Source, "Leaderboard refresh already queued")
	}
}

func (p *Plugin) lookupLeaderboard(ctx context.Context, event string) (*Leaderboard, error) {
	if event == "" {
		now := time.Now().In(tzEastern)
		year := now.Year()
		if now.Month() == time.December {
			event = strconv.Itoa(year)
		} else {
			event = strconv.Itoa(year - 1)
		}
	}

	leaderboard, err := p.aocClient.GetLeaderboard(ctx, event, p.config.AOCLeaderboard)
	if err != nil {
		return nil, err
	}

	return leaderboard, nil
}

func (p *Plugin) readLastUpdated() (time.Time, error) {
	data, err := os.ReadFile(p.config.TimestampFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, nil
		}

		return time.Time{}, err
	}

	timestamp, err := strconv.ParseInt(string(bytes.TrimSpace(data)), 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

func (p *Plugin) writeLastUpdated(cur time.Time) error {
	return os.WriteFile(p.config.TimestampFile, []byte(strconv.FormatInt(cur.Unix(), 10)), fs.ModePerm)
}

func (p *Plugin) updateLeaderboard(ctx context.Context) error {
	leaderboard, err := p.lookupLeaderboard(ctx, "")
	if err != nil {
		return err
	}

	// Update the cache with the latest leaderboard
	p.cacheLock.Lock()
	p.cachedLeaderboard = leaderboard
	p.cacheLock.Unlock()

	lastUpdated, err := p.readLastUpdated()
	if err != nil {
		return err
	}

	rawEvents := leaderboard.GetEvents()
	p.logger.With(slog.Int("total_event_count", len(rawEvents))).Debug("Found events")

	var events []*Event
	for _, event := range rawEvents {
		if !event.Timestamp.After(lastUpdated) {
			continue
		}

		events = append(events, event)
	}
	eventCount := len(events)
	if eventCount == 0 {
		p.logger.With(
			slog.Int("total_event_count", len(rawEvents)),
			slog.Int("filtered_event_count", eventCount),
		).Info("No new events")
	} else {
		p.logger.With(
			slog.Int("event_count", len(rawEvents)),
			slog.Int("filtered_event_count", eventCount),
		).Info("Found new events")
	}

	for _, event := range events {
		p.sbClient.Inner.SendMessage(ctx, &pb.SendMessageRequest{
			ChannelId: p.config.AOCChannel,
			Text:      event.String(),
		})

		// It's arguably worse to cause a write on every message sent, but this
		// will make it possible to properly handle things if we fail to send a
		// message without having to start over.
		p.writeLastUpdated(event.Timestamp)
	}

	return nil
}

func (p *Plugin) runAOCUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(leaderboardTickFrequency)
	defer ticker.Stop()

	p.logger.Info("Running initial leaderboard update")

	// Queue an update on startup. In the future, we may want to drop this in
	// favor of manual updates.
	err := p.updateLeaderboard(ctx)
	if err != nil {
		p.logger.With(slog.Any("error", err)).Error(fmt.Sprintf("Failed to update leaderboard, trying again in %s", leaderboardTickFrequency))
	}

	for {
		select {
		case <-ticker.C:
			p.logger.Info("Running scheduled leaderboard update")

			err = p.updateLeaderboard(ctx)
			if err != nil {
				p.logger.With(slog.Any("error", err)).Error(fmt.Sprintf("Failed to update leaderboard, trying again in %s", leaderboardTickFrequency))
			}
		case updateResp := <-p.queueUpdate:
			// If we got a request to queue an update, reset the 15m timer to
			// avoid purposefully making requests too frequently. It is still
			// technically possible for users to spam this, but hopefully that
			// won't be a major issue.
			p.logger.Info("Running manual leaderboard update")

			err = p.updateLeaderboard(ctx)
			if err != nil {
				p.logger.With(slog.Any("error", err)).Error(fmt.Sprintf("Failed to update leaderboard, trying again in %s", leaderboardTickFrequency))
			}

			p.logger.Info(fmt.Sprintf("Resetting leaderboard update ticker to %s", leaderboardTickFrequency))

			ticker.Reset(leaderboardTickFrequency)

			// Notify the listener that we're done
			select {
			case updateResp <- err == nil:
			default:
			}
		}
	}
}

func calculateNextReminder(leaderboard *Leaderboard, now time.Time) (time.Time, int) {
	day1Start := time.Unix(leaderboard.Day1Timestamp, 0).In(tzEastern)
	firstNotificationTime := day1Start.Add(-15 * time.Minute)

	// If the first notification time hasn't been hit, we're some time before the event has started, so we can just
	// return the firstNotificationTime.
	if firstNotificationTime.After(now) {
		return firstNotificationTime, 1
	}

	// Calculate how many days before or after the start date we are
	daysSinceStart := int(now.Sub(firstNotificationTime).Hours() / 24) + 1

	// If we're after the last notification, we need to send what we think the first notification for next year will be.
	if daysSinceStart >= leaderboard.NumDays {
		return firstNotificationTime.AddDate(1, 0, 0), 1
	}

	// During the event we add which day we're on to the first notification time
	return firstNotificationTime.AddDate(0, 0, daysSinceStart), daysSinceStart + 1
}

func (p *Plugin) scheduleReminders(ctx context.Context) {
	for {
		// Fetch the current leaderboard to get Day1Timestamp and NumDays
		leaderboard, err := p.lookupLeaderboard(ctx, "")
		if err != nil {
			p.logger.With(slog.Any("error", err)).Error("Failed to lookup leaderboard for reminders, retrying in 1 hour")
			select {
			case <-time.After(1 * time.Hour):
				continue
			case <-ctx.Done():
				return
			}
		}

		// Determine the next notification time
		now := time.Now().In(tzEastern)
		nextReminder, day := calculateNextReminder(leaderboard, now)

		sleepDuration := nextReminder.Sub(now)
		p.logger.With(
			slog.Time("next_reminder", nextReminder),
			slog.Duration("sleep_duration", sleepDuration),
			slog.Int("day", day),
		).Info("Next reminder scheduled")

		select {
		case <-time.After(sleepDuration):
			msg := fmt.Sprintf("Advent of Code day %d is starting in 15 minutes!", day)
			_, err := p.sbClient.Inner.SendMessage(ctx, &pb.SendMessageRequest{
				ChannelId: p.config.AOCChannel,
				Text:      msg,
			})
			if err != nil {
				p.logger.With(slog.Any("error", err)).Error("Failed to send reminder message")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *Plugin) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	// NOTE: we only actually care if the seabird stream dies - the AOC Update
	// Loop can fail in the background, and we hope it will recover eventually.

	group.Go(func() error {
		return p.runSeabirdStream(ctx)
	})

	group.Go(func() error {
		p.runAOCUpdateLoop(ctx)
		return nil
	})

	group.Go(func() error {
		p.scheduleReminders(ctx)
		return nil
	})

	return group.Wait()
}
