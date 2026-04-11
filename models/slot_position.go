package models

import "time"

type SlotPosition struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SlotID    uint      `gorm:"index;not null" json:"slotId"`
	Slot      Slot      `gorm:"foreignKey:SlotID" json:"-"`
	Timestamp time.Time `gorm:"index;not null" json:"timestamp"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
}
