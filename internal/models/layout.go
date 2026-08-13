package models

import "time"

type EventFloorLayout struct {
	ID         int64
	EventID    int64
	LayoutJSON string
	RowVersion int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type StandaloneFloorLayout struct {
	ID              int64
	Name            string
	Venue           string
	GuestCount      int
	WaiterCount     int
	WaiterNamesJSON string
	LayoutJSON      string
	Active          bool
	RowVersion      int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type FloorLayoutDocument struct {
	Version  int                  `json:"version"`
	Width    int                  `json:"width"`
	Height   int                  `json:"height"`
	Waiters  []string             `json:"waiters,omitempty"`
	Elements []FloorLayoutElement `json:"elements"`
}

type FloorLayoutElement struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation,omitempty"`
	Label    string  `json:"label,omitempty"`
	Waiter   string  `json:"waiter,omitempty"`
	Color    string  `json:"color,omitempty"`
	Seats    int     `json:"seats,omitempty"`
	ZIndex   int     `json:"zIndex,omitempty"`
}
