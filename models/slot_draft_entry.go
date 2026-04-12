package models

type SlotDraftEntry struct {
	ID                 uint         `gorm:"primaryKey" json:"id"`
	SlotRevisionID     uint         `gorm:"index;not null" json:"slotRevisionId"`
	DepartureAirportID uint         `gorm:"not null" json:"departureAirportId"`
	DepartureAirport   Airport      `gorm:"foreignKey:DepartureAirportID" json:"departureAirport,omitempty"`
	DepRouteID         uint         `gorm:"not null" json:"depRouteId"`
	DepRoute           RouteSegment `gorm:"foreignKey:DepRouteID" json:"depRoute,omitempty"`
	TrackID            uint         `gorm:"not null" json:"trackId"`
	Track              RouteSegment `gorm:"foreignKey:TrackID" json:"track,omitempty"`
	ArrRouteID         uint         `gorm:"not null" json:"arrRouteId"`
	ArrRoute           RouteSegment `gorm:"foreignKey:ArrRouteID" json:"arrRoute,omitempty"`
	ArrivalAirportID   uint         `gorm:"not null" json:"arrivalAirportId"`
	ArrivalAirport     Airport      `gorm:"foreignKey:ArrivalAirportID" json:"arrivalAirport,omitempty"`
	SlotCount          uint         `gorm:"not null" json:"slotCount"`
}
