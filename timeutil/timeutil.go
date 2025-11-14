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
