package models

import "time"

type Slot struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	SlotRevisionID       uint           `gorm:"index;not null" json:"slotRevisionId"`
	DepartureTime        time.Time      `json:"departureTime"`
	ProjectedArrivalTime time.Time      `json:"projectedArrivalTime"`
	DepartureAirportID   uint           `json:"departureAirportId"`
	DepartureAirport     Airport        `gorm:"foreignKey:DepartureAirportID" json:"departureAirport,omitempty"`
	ArrivalAirportID     uint           `json:"arrivalAirportId"`
	ArrivalAirport       Airport        `gorm:"foreignKey:ArrivalAirportID" json:"arrivalAirport,omitempty"`
	RouteSegments        []RouteSegment `gorm:"many2many:slot_route_segments;joinForeignKey:SlotID;joinReferences:RouteSegmentID" json:"routeSegments,omitempty"`
}
