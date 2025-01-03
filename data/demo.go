package data

import "time"

const demo = -12 * time.Hour // for demo, default 0

func Now() time.Time {
	return time.Now().UTC().Add(demo)
}

func DateNow() time.Time {
	return Now().Truncate(24 * time.Hour)
}
