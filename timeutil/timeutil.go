package timeutil

import (
	"math/rand"
	"time"
)

func IsBetween(t time.Time, start, end time.Time) bool {
	return t.Equal(start) || t.After(start) && t.Before(end) || t.Equal(end)
}

func RandomSleep() {
	rand.NewSource(time.Now().UTC().UnixNano())
	n := rand.Intn(10)
	time.Sleep(time.Duration(n) * time.Second)
}
