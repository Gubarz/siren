package gui

import (
	"sliver-gui/internal/journal"
)

type JournalPage struct {
	Entries []journal.Entry `json:"entries"`
	Total   int             `json:"total"`
}

type JournalTimeSeriesResponse struct {
	Buckets []journal.TimeBucket `json:"buckets"`
}

func (a *App) QueryJournal(filter journal.Filter) (JournalPage, error) {
	entries, total, err := a.Journal.Query(a.ctx, filter)
	if err != nil {
		return JournalPage{}, err
	}
	return JournalPage{Entries: entries, Total: total}, nil
}

func (a *App) GetJournalVerbCounts(filter journal.Filter) (map[string]int64, error) {
	return a.Journal.VerbCounts(a.ctx, filter)
}

func (a *App) GetJournalTimeSeries(filter journal.TimeSeriesFilter) (JournalTimeSeriesResponse, error) {
	buckets, err := a.Journal.TimeSeries(a.ctx, filter)
	if err != nil {
		return JournalTimeSeriesResponse{}, err
	}
	return JournalTimeSeriesResponse{Buckets: buckets}, nil
}
