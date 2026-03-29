package models

type Sector struct {
	ThroughputPoint
	SectorBoundaries []SectorBoundary `gorm:"foreignKey:SectorID;constraint:OnDelete:CASCADE;" json:"sectorBoundaries,omitempty"`
	EventID          *uint            `gorm:"index" json:"eventId,omitempty"`
}

type SectorBoundary struct {
	ID           uint                      `gorm:"primaryKey" json:"id"`
	SectorID     uint                      `gorm:"index;not null" json:"sectorId"`
	MaxLatitude  float64                   `json:"maxLatitude"`
	MinLatitude  float64                   `json:"minLatitude"`
	MaxLongitude float64                   `json:"maxLongitude"`
	MinLongitude float64                   `json:"minLongitude"`
	Coordinates  []SectorBoundaryCoordinate `gorm:"foreignKey:SectorBoundaryID;constraint:OnDelete:CASCADE;" json:"coordinates,omitempty"`
}

type SectorBoundaryCoordinate struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	SectorBoundaryID uint    `gorm:"index;not null" json:"sectorBoundaryId"`
	SortOrder        uint    `gorm:"not null" json:"sortOrder"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
}
