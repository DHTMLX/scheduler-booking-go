package data

import "time"

func Now() time.Time {
	return time.Now().UTC()
}

func DateNow() time.Time {
	return Now().Truncate(24 * time.Hour)
}
