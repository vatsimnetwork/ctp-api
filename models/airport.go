package models

type Airport struct {
	ThroughputPoint
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	NumberOfVotes uint16  `json:"numberOfVotes"`
	EventID       uint    `gorm:"index;not null" json:"eventId"`
}
