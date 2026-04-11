package models

type DeparturePairPreference struct {
	ID                 uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID            uint    `gorm:"not null" json:"eventId"`
	DepartureAirportID uint    `gorm:"not null" json:"departureAirportId"`
	DepartureAirport   Airport `gorm:"foreignKey:DepartureAirportID" json:"departureAirport,omitempty"`
	ArrivalAirportID   uint    `gorm:"not null" json:"arrivalAirportId"`
	ArrivalAirport     Airport `gorm:"foreignKey:ArrivalAirportID" json:"arrivalAirport,omitempty"`
	Preference         uint8   `gorm:"not null;default:0" json:"preference"`
}

const (
	PreferenceNone      uint8 = 0
	PreferenceDeferred  uint8 = 1
	PreferencePreferred uint8 = 2
)
