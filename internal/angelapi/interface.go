package angelapi

import "time"

var timeFormatISO8601 = "2006-01-02T15:04:05.999Z"

// Service defines an angelapi service.
type Service interface {
	ListLocations(*ListLocationsOpts) ([]Location, error)
	ListShiftsInLocation(int64, *ListShiftsInLocationOpts) ([]Shift, error)
}

// ListLocationsOpts holds options for listing locations.
type ListLocationsOpts struct {
}

// ListShiftsInLocationOpts holds options for listing shifts in a location.
type ListShiftsInLocationOpts struct {
}

// Location represents a location.
type Location struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Shift represents a shift.
type Shift struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartsAtRaw string `json:"starts_at"` // ISO-8601.
	EndsAtRaw   string `json:"ends_at"`   // ISO-8601.
	StartsAt    time.Time
	EndsAt      time.Time
	Entries     []ShiftEntry `json:"entries"`
}

// ShiftType represents a shift type.
type ShiftType struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ShiftEntry represents a shift entry.
type ShiftEntry struct {
	Users []User    `json:"users"`
	Type  ShiftType `json:"type"`
	Needs int64     `json:"needs"`
}

// User represents a user.
type User struct {
	ID        int64       `json:"id"`
	NickName  string      `json:"name"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	Pronoun   string      `json:"pronoun"`
	Contact   UserContact `json:"contact"`
}

// UserContact represents a user contact.
type UserContact struct {
	DECT   string `json:"dect"`
	Mobile string `json:"mobile"`
}
