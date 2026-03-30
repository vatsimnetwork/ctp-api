package models

type EventTag struct {
	ID                     uint    `gorm:"primaryKey" json:"id"`
	EventID                uint    `gorm:"uniqueIndex:idx_event_tag_name;index;not null" json:"eventId"`
	Name                   string  `gorm:"uniqueIndex:idx_event_tag_name;not null" json:"name"`
	MaximumAircraftPerHour *uint16 `json:"maximumAircraftPerHour,omitempty"`
}
