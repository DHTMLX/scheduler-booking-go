package common

import (
	"testing"
)

func Test_m2t(t *testing.T) {
	cases := []struct {
		minutes       int
		expectedHours string
	}{
		{
			minutes:       15,
			expectedHours: "00:15",
		},
		{
			minutes:       30,
			expectedHours: "00:30",
		},
		{
			minutes:       60,
			expectedHours: "01:00",
		},
		{
			minutes:       90,
			expectedHours: "01:30",
		},
		{
			minutes:       135,
			expectedHours: "02:15",
		},
		{
			minutes:       545,
			expectedHours: "09:05",
		},
		{
			minutes:       875,
			expectedHours: "14:35",
		},
		{
			minutes:       1020,
			expectedHours: "17:00",
		},
		{
			minutes:       1260,
			expectedHours: "21:00",
		},
		{
			minutes:       1440,
			expectedHours: "24:00",
		},
		{
			minutes:       1480,
			expectedHours: "24:40",
		},
	}

	for _, c := range cases {
		hours := m2t(c.minutes)
		if hours != c.expectedHours {
			t.Fatalf("expected %s, got %s", c.expectedHours, hours)
		}
	}
}
