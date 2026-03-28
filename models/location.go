package models

type Location struct {
	ThroughputPoint
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	WaypointID     *int64   `json:"waypointId,omitempty"`
	EventID        *uint    `gorm:"index" json:"eventId,omitempty"`
	RouteSegmentID *uint    `gorm:"index" json:"routeSegmentId,omitempty"`
	SortOrder      uint     `gorm:"default:0" json:"sortOrder"`
}
