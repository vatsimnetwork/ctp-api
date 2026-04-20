package models

import "time"

// BookingSlotMapping records the association between a booking-portal CSV row ID
// and the internal slot that was matched for it during a booking import.
type BookingSlotMapping struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	EventID        uint      `gorm:"index;not null" json:"eventId"`
	SlotRevisionID uint      `gorm:"index;not null" json:"slotRevisionId"`
	BookingID      uint      `gorm:"not null" json:"bookingId"`
	SlotID         uint      `gorm:"not null" json:"slotId"`
	Slot           Slot      `gorm:"foreignKey:SlotID" json:"slot,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}
