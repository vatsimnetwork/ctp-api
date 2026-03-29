package models

type RouteSegment struct {
	ThroughputPoint
	RouteString                 string             `json:"routeString"`
	RouteSegmentGroup           string             `json:"routeSegmentGroup"`
	Color                       string             `json:"color"`
	Enabled                     bool               `gorm:"default:true" json:"enabled"`
	Facilities                  string             `json:"facilities"`
	Tags                        []RouteSegmentTag  `gorm:"foreignKey:RouteSegmentID;constraint:OnDelete:CASCADE;" json:"tags,omitempty"`
	ProvidedFacilityProgression []Sector           `gorm:"many2many:route_segment_sectors;" json:"providedFacilityProgression,omitempty"`
	Slots                       []Slot             `gorm:"many2many:slot_route_segments;" json:"-"`
	Locations                   []Location         `gorm:"foreignKey:RouteSegmentID;constraint:OnDelete:CASCADE;" json:"locations,omitempty"`
	RouteRevision               uint               `json:"routeRevision"`
	EventID                     *uint              `gorm:"index" json:"eventId,omitempty"`
}

type RouteSegmentTag struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	RouteSegmentID uint   `gorm:"index;not null" json:"routeSegmentId"`
	Tag            string `gorm:"not null" json:"tag"`
}
