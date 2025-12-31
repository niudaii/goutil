package timeutil

import (
	"math/rand"
	"time"
)

func IsBetween(t time.Time, start, end time.Time) bool {
	return t.Equal(start) || t.After(start) && t.Before(end) || t.Equal(end)
}

func RandomSleep(max int) {
	rand.NewSource(time.Now().UTC().UnixNano())
	n := rand.Intn(max)
	time.Sleep(time.Duration(n) * time.Second)
}

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func RandomJitterSleep(minMs, maxMs int) {
	randomMs := rand.Intn(maxMs-minMs+1) + minMs

	time.Sleep(time.Duration(randomMs) * time.Millisecond)
}

func FormatTimeToShanghai(t *time.Time) string {
	if t == nil {
		return ""
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
	return t.In(loc).Format(time.DateTime)
}

func FormatDateToShanghai(t *time.Time) string {
	if t == nil {
		return ""
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
	return t.In(loc).Format(time.DateOnly)
}
