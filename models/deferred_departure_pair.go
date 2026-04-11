package models

type DeferredDeparturePair struct {
	ID                 uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID            uint    `gorm:"not null;uniqueIndex:idx_deferred_pair" json:"eventId"`
	DepartureAirportID uint    `gorm:"not null;uniqueIndex:idx_deferred_pair" json:"departureAirportId"`
	DepartureAirport   Airport `gorm:"foreignKey:DepartureAirportID" json:"departureAirport,omitempty"`
	ArrivalAirportID   uint    `gorm:"not null;uniqueIndex:idx_deferred_pair" json:"arrivalAirportId"`
	ArrivalAirport     Airport `gorm:"foreignKey:ArrivalAirportID" json:"arrivalAirport,omitempty"`
}
