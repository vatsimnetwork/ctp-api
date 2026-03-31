package models

// SlotRouteSegment is the explicit join model for the slot_route_segments table.
// The Order field records the position of the route segment within the slot's route
// (0 = departure route, 1 = track, 2 = arrival route).
type SlotRouteSegment struct {
	SlotID         uint `gorm:"primaryKey;autoIncrement:false" json:"slotId"`
	RouteSegmentID uint `gorm:"primaryKey;autoIncrement:false" json:"routeSegmentId"`
	Order          uint `gorm:"column:order;not null;default:0" json:"order"`
}

func (SlotRouteSegment) TableName() string { return "slot_route_segments" }
