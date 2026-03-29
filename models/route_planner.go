package models

import "time"

type Airway struct {
	Identifier string           `gorm:"primaryKey" json:"identifier"`
	Waypoints  []AirwayWaypoint `gorm:"foreignKey:AirwayID;constraint:OnDelete:CASCADE;" json:"waypoints,omitempty"`
}

type AirwayWaypoint struct {
	ID         uint     `gorm:"primaryKey" json:"id"`
	AirwayID   string   `gorm:"index;not null" json:"airwayId"`
	LocationID uint     `gorm:"not null" json:"locationId"`
	Location   Location `json:"location,omitempty"`
	Order      uint     `gorm:"not null" json:"order"`
}

type CustomFix struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Identifier string    `gorm:"uniqueIndex;not null" json:"identifier"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
}

type HighlightedWaypoint struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Identifier string    `gorm:"uniqueIndex;not null" json:"identifier"`
	Color      string    `gorm:"default:'#f97316'" json:"color"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
	WaypointID *int64    `gorm:"index" json:"waypointId,omitempty"`
	Waypoint   *Waypoint `gorm:"foreignKey:WaypointID" json:"waypoint,omitempty"`
}

type RouteRevisionSet struct {
	ID        uint                 `gorm:"primaryKey" json:"id"`
	Number    uint                 `gorm:"uniqueIndex;not null" json:"number"`
	CreatedAt time.Time            `json:"createdAt"`
	Entries   []RouteRevisionEntry `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE;" json:"entries,omitempty"`
}

type RouteRevisionEntry struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	RevisionID     uint   `gorm:"index;not null" json:"revisionId"`
	Identifier     string `gorm:"not null" json:"identifier"`
	Group          string `json:"group"`
	RouteString    string `json:"routeString"`
	Facilities     string `json:"facilities"`
	Tags           string `json:"tags"`
	Color          string `json:"color"`
	Enabled        bool   `gorm:"default:true" json:"enabled"`
}
