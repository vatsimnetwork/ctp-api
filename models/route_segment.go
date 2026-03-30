package models

import "encoding/json"

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
	ID             uint     `gorm:"primaryKey" json:"id"`
	RouteSegmentID uint     `gorm:"index;not null" json:"routeSegmentId"`
	TagID          *uint    `gorm:"index" json:"-"`
	TagRef         EventTag `gorm:"foreignKey:TagID" json:"-"`
}

func (r RouteSegmentTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID                     uint    `json:"id"`
		RouteSegmentID         uint    `json:"routeSegmentId"`
		Tag                    string  `json:"tag"`
		MaximumAircraftPerHour *uint16 `json:"maximumAircraftPerHour,omitempty"`
	}{
		ID:                     r.ID,
		RouteSegmentID:         r.RouteSegmentID,
		Tag:                    r.TagRef.Name,
		MaximumAircraftPerHour: r.TagRef.MaximumAircraftPerHour,
	})
}
