package service

import (
	"reflect"
	"testing"
)

func Test_m2t(t *testing.T) {
	cases := []struct {
		minutes       int
		expectedHours string
	}{
		{
			minutes:       15,
			expectedHours: "0:15",
		},
		{
			minutes:       30,
			expectedHours: "0:30",
		},
		{
			minutes:       60,
			expectedHours: "1:00",
		},
		{
			minutes:       90,
			expectedHours: "1:30",
		},
		{
			minutes:       135,
			expectedHours: "2:15",
		},
		{
			minutes:       545,
			expectedHours: "9:05",
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
			minutes:       1600,
			expectedHours: "24:00",
		},
	}

	for _, c := range cases {
		h := m2t(c.minutes)
		if h != c.expectedHours {
			t.Fatalf("expected %s, got %s", c.expectedHours, h)
		}
	}
}

func TestDaysFromRules(t *testing.T) {
	cases := []struct {
		rrule string
		days  []int
	}{
		{
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO",
			days:  []int{1},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FR",
			days:  []int{1, 3, 5},
		},
		{
			rrule: "BYDAY=SA,SU;FREQ=WEEKLY;INTERVAL=1",
			days:  []int{6, 0},
		},
		{
			rrule: "BYDAY=SU;INTERVAL=2;FREQ=WEEKLY",
			days:  []int{0},
		},
		{
			rrule: "FREQ=WEEKLY;BYDAY=TU,TH;INTERVAL=1",
			days:  []int{2, 4},
		},
		{
			rrule: "INTERVAL=1;BYDAY=TU,TH;FREQ=WEEKLY",
			days:  []int{2, 4},
		},
		{
			rrule: "INTERVAL=1;BYDAY=;FREQ=WEEKLY",
			days:  []int{},
		},
		{
			rrule: "INTERVAL=1;BYDAY=",
			days:  []int{},
		},
		{
			rrule: "BYDAY=;FREQ=WEEKLY",
			days:  []int{},
		},
		{
			rrule: "BYDAY=;",
			days:  []int{},
		},
		{
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=Mo,wE,fr",
			days:  []int{1, 3, 5},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FE",
			days:  []int{1, 3},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=FR,WE,MO",
			days:  []int{5, 3, 1},
		},
	}

	for _, c := range cases {
		days := daysFromRules(c.rrule)
		if !reflect.DeepEqual(c.days, days) {
			t.Fatalf("expected %v, got %v", c.days, days)
		}
	}
}
