package models

import "time"

type Airport struct {
	ID                       uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	WaypointID               int64      `gorm:"not null;uniqueIndex:idx_airport_event" json:"waypointId"`
	Waypoint                 Waypoint   `gorm:"foreignKey:WaypointID" json:"waypoint,omitempty"`
	EventID                  uint       `gorm:"not null;index;uniqueIndex:idx_airport_event" json:"eventId"`
	MaximumAircraftPerHour   uint16     `gorm:"default:20" json:"maximumAircraftPerHour"`
	MaximumSlots             uint16     `gorm:"default:65535" json:"maximumSlots"`
	SlotsAllocated           uint16     `json:"slotsAllocated"`
	NumberOfVotes            uint16     `json:"numberOfVotes"`
	DepartureTimeWindowStart *time.Time `json:"departureTimeWindowStart,omitempty"`
	EarliestArrivalTime      *time.Time `json:"earliestArrivalTime,omitempty"`
}
