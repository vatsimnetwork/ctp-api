package models

import "time"

type SlotRevision struct {
	ID                             uint                 `gorm:"primaryKey" json:"id"`
	EventID                        uint                 `gorm:"index;not null" json:"eventId"`
	Number                         uint                 `gorm:"not null" json:"number"`
	SlotGenerationOutputCommentary string               `json:"slotGenerationOutputCommentary"`
	SimulationOutputCommentary     string               `json:"simulationOutputCommentary"`
	CreatedAt                            time.Time                          `json:"createdAt"`
	Slots                                []Slot                             `gorm:"foreignKey:SlotRevisionID;constraint:OnDelete:CASCADE;" json:"slots,omitempty"`
	ThroughputStates                     []ThroughputState                  `gorm:"foreignKey:SlotRevisionID;constraint:OnDelete:CASCADE;" json:"throughputStates,omitempty"`
	ThroughputSnapshots                  []ThroughputSnapshot               `gorm:"foreignKey:SlotRevisionID;constraint:OnDelete:CASCADE;" json:"throughputSnapshots,omitempty"`
	SlotDraftEntries                     []SlotDraftEntry                   `gorm:"foreignKey:SlotRevisionID;constraint:OnDelete:CASCADE;" json:"slotDraftEntries,omitempty"`
	AirportPairDepartureWindowShifts     []AirportPairDepartureWindowShift  `gorm:"foreignKey:SlotRevisionID;constraint:OnDelete:CASCADE;" json:"airportPairDepartureWindowShifts,omitempty"`
}
