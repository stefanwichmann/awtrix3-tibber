package main

import (
	"os"
	"time"
)

func lookupEnv(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func durationUntilNextFullHour() time.Duration {
	now := time.Now()
	nextFullHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())
	nextFullHour = nextFullHour.Add(1 * time.Hour)
	return time.Until(nextFullHour)
}
