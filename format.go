package main

import "time"

const isoMillisLayout = "2006-01-02T15:04:05.000Z"

// jsISO renders a timestamp the same way JavaScript's Date.toISOString does.
func jsISO(t time.Time) string {
	return t.UTC().Format(isoMillisLayout)
}

func nowISO() string {
	return jsISO(time.Now())
}

func parseISODate(value string) (time.Time, bool) {
	if len(value) != 10 {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	if parsed.Format("2006-01-02") != value {
		return time.Time{}, false
	}
	return parsed, true
}

func isISODate(value string) bool {
	_, ok := parseISODate(value)
	return ok
}

func todayISO() string {
	return time.Now().Format("2006-01-02")
}

func addDaysISO(value string, days int) string {
	parsed, ok := parseISODate(value)
	if !ok {
		return value
	}
	noon := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 12, 0, 0, 0, time.Local)
	return noon.AddDate(0, 0, days).Format("2006-01-02")
}

func startOfLocalDay(value string) string {
	parsed, ok := parseISODate(value)
	if !ok {
		return value
	}
	return jsISO(parsed)
}

func endOfLocalDay(value string) string {
	parsed, ok := parseISODate(value)
	if !ok {
		return value
	}
	next := parsed.AddDate(0, 0, 1)
	return jsISO(next.Add(-time.Millisecond))
}
