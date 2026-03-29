package models

type Waypoint struct {
	ID                     int64   `gorm:"primaryKey" json:"id"`
	Identifier             string  `gorm:"not null;index" json:"identifier"`
	Latitude               float64 `gorm:"not null" json:"latitude"`
	Longitude              float64 `gorm:"not null" json:"longitude"`
	MaximumAircraftPerHour uint16  `gorm:"default:20" json:"maximumAircraftPerHour"`
	MaximumSlots           uint16  `json:"maximumSlots"`
}
