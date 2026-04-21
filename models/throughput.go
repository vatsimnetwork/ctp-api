package models

import "time"

type ThroughputPoint struct {
	ID                     uint   `gorm:"primaryKey" json:"id"`
	Identifier             string `gorm:"not null;uniqueIndex" json:"identifier"`
	MaximumAircraftPerHour uint16 `gorm:"default:65535" json:"maximumAircraftPerHour"`
}

type ThroughputState struct {
	ID                       uint       `gorm:"primaryKey" json:"id"`
	SlotRevisionID           uint       `gorm:"index;not null" json:"slotRevisionId"`
	ThroughputPointType      string     `gorm:"not null;index" json:"throughputPointType"`
	ThroughputPointID        int64      `gorm:"not null;index" json:"throughputPointId"`
	MaximumSlots             uint16     `json:"maximumSlots"`
	SlotsAllocated           uint16     `json:"slotsAllocated"`
	DepartureTimeWindowStart *time.Time `json:"departureTimeWindowStart,omitempty"`
}

type ThroughputSnapshot struct {
	ID                  uint   `gorm:"primaryKey" json:"id"`
	SlotRevisionID      uint   `gorm:"index;not null" json:"slotRevisionId"`
	ThroughputPointType string `gorm:"not null" json:"throughputPointType"`
	ThroughputPointID   int64  `gorm:"not null" json:"throughputPointId"`
	MinuteOffset        int    `gorm:"not null" json:"minuteOffset"`
	SlotID              uint   `gorm:"not null" json:"slotId"`
	Slot                Slot   `gorm:"foreignKey:SlotID" json:"slot,omitempty"`
}
