package handlers

import (
	"sort"
	"time"

	"buffetflow/internal/models"
)

// buildEventCalendar keeps calendar calculations in the server, which makes
// the same ordering and availability information available to a cached PWA
// page without relying on a third-party calendar script.
func buildEventCalendar(events []models.Event, rawMonth, selected string, location *time.Location) models.EventCalendar {
	now := time.Now().In(location)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	if parsed, err := time.ParseInLocation("2006-01", rawMonth, location); err == nil {
		month = time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, location)
	}
	if selected == "" {
		if now.Year() == month.Year() && now.Month() == month.Month() {
			selected = now.Format("2006-01-02")
		} else {
			selected = month.Format("2006-01-02")
		}
	}

	calendar := models.EventCalendar{
		Month:         month,
		PreviousMonth: month.AddDate(0, -1, 0).Format("2006-01"),
		NextMonth:     month.AddDate(0, 1, 0).Format("2006-01"),
		SelectedDate:  selected,
	}
	byDay := make(map[string][]models.Event, len(events))
	for _, event := range events {
		localStart := event.StartsAt.In(location)
		key := localStart.Format("2006-01-02")
		byDay[key] = append(byDay[key], event)
	}

	// Monday is the first column, which matches the customary Brazilian agenda.
	start := month.AddDate(0, 0, -((int(month.Weekday()) + 6) % 7))
	today := now.Format("2006-01-02")
	for index := 0; index < 42; index++ {
		date := start.AddDate(0, 0, index)
		key := date.Format("2006-01-02")
		calendar.Days = append(calendar.Days, models.CalendarDay{
			Date: date, DateKey: key, Day: date.Day(), InMonth: date.Month() == month.Month(),
			IsToday: key == today, Events: byDay[key],
		})
	}
	if calendar.SelectedDate != "" {
		calendar.SelectedEvents = append(calendar.SelectedEvents, byDay[calendar.SelectedDate]...)
	}

	upcomingByDay := map[string][]models.Event{}
	for _, event := range events {
		if event.StartsAt.In(location).Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)) {
			continue
		}
		key := event.StartsAt.In(location).Format("2006-01-02")
		upcomingByDay[key] = append(upcomingByDay[key], event)
	}
	keys := make([]string, 0, len(upcomingByDay))
	for key := range upcomingByDay {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		date, _ := time.ParseInLocation("2006-01-02", key, location)
		calendar.Upcoming = append(calendar.Upcoming, models.EventDateGroup{Date: date, Events: upcomingByDay[key]})
	}
	return calendar
}
