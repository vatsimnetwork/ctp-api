package models

type Location struct {
	ID             uint     `gorm:"primaryKey;autoIncrement" json:"id"`
	RouteSegmentID uint     `gorm:"not null;index" json:"routeSegmentId"`
	WaypointID     int64    `gorm:"not null;index" json:"waypointId"`
	Waypoint       Waypoint `gorm:"foreignKey:WaypointID" json:"waypoint,omitempty"`
	SortOrder      uint     `gorm:"default:0" json:"sortOrder"`
}
