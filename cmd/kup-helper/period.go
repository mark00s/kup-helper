package main

import "time"

// startFromDayOfMonth returns the given day of the month preceding now's
// month. If that month is shorter (e.g. day 31 and a 30-day month), the last
// day of that month is used.
func startFromDayOfMonth(now time.Time, day int) time.Time {
	prevMonthFirst := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	lastDay := prevMonthFirst.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(prevMonthFirst.Year(), prevMonthFirst.Month(), day, 0, 0, 0, 0, now.Location())
}
