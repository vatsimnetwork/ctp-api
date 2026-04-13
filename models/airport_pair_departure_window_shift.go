package models

type AirportPairDepartureWindowShift struct {
	ID                 uint    `gorm:"primaryKey" json:"id"`
	SlotRevisionID     uint    `gorm:"index;not null;uniqueIndex:uq_window_shift_rev_pair,priority:1" json:"slotRevisionId"`
	DepartureAirportID uint    `gorm:"not null;uniqueIndex:uq_window_shift_rev_pair,priority:2" json:"departureAirportId"`
	DepartureAirport   Airport `gorm:"foreignKey:DepartureAirportID" json:"departureAirport,omitempty"`
	ArrivalAirportID   uint    `gorm:"not null;uniqueIndex:uq_window_shift_rev_pair,priority:3" json:"arrivalAirportId"`
	ArrivalAirport     Airport `gorm:"foreignKey:ArrivalAirportID" json:"arrivalAirport,omitempty"`
	StartShiftHours    float64 `gorm:"not null;default:0" json:"startShiftHours"`
	EndShiftHours      float64 `gorm:"not null;default:0" json:"endShiftHours"`
}
