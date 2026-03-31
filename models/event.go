package models

import "time"

type VATSIMEvent struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Title               string    `gorm:"not null" json:"title"`
	RouteRevision       uint      `json:"routeRevision"`
	Date                time.Time `gorm:"type:date" json:"date"`
	DepartureTimeWindow Duration  `gorm:"default:10800000000000" json:"departureTimeWindow" swaggertype:"string" example:"3h0m0s"`

	IntendedSlotGenerationMode                            SlotGenerationMode                        `gorm:"default:0" json:"intendedSlotGenerationMode"`
	DepartureTimeWindowOffsetSynchronizationLongitude     float64                                   `gorm:"default:-30" json:"departureTimeWindowOffsetSynchronizationLongitude"`
	SimulationAnalysisResolutionInMinutes                 uint                                      `gorm:"default:2" json:"simulationAnalysisResolutionInMinutes"`
	ShouldSimulationUseActualWeatherForecastData          bool                                      `json:"shouldSimulationUseActualWeatherForecastData"`
	IntendedDepartureTimeWindowOffsetsCalculationMode     DepartureTimeWindowOffsetsCalculationMode `gorm:"default:1" json:"intendedDepartureTimeWindowOffsetsCalculationMode"`
	DepartureTimeWindowOffsetSynchronizationTimeOfDay     string                                    `gorm:"default:'16:00'" json:"departureTimeWindowOffsetSynchronizationTimeOfDay"`
	CalculateThroughputDataOnlyForManuallyProvidedSectors bool                                      `gorm:"default:true" json:"calculateThroughputDataOnlyForManuallyProvidedSectors"`
	IntendedWaypointThroughputCalculationMode             WaypointThroughputCalculationMode         `gorm:"default:1" json:"intendedWaypointThroughputCalculationMode"`
	ThresholdToCheckIfAirplaneIsCountedAtWaypointInNm     float64                                   `gorm:"default:5.0" json:"thresholdToCheckIfAirplaneIsCountedAtWaypointInNm"`
	CalculationFallbackGroundSpeed                        float64                                   `gorm:"default:300.0" json:"calculationFallbackGroundSpeed"`
	HighSimulationAccuracy                                bool                                      `json:"highSimulationAccuracy"`

	Airports      []Airport      `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE;" json:"airports,omitempty"`
	RouteSegments []RouteSegment `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE;" json:"routeSegments,omitempty"`
	Sectors       []Sector       `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE;" json:"sectors,omitempty"`
	SlotRevisions []SlotRevision `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE;" json:"slotRevisions,omitempty"`
}
