package handlers

import (
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// GetSlotPositions godoc
//
//	@Summary	Get slot positions at a specific timestamp
//	@Tags		slots
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id			path		int		true	"Event ID"
//	@Param		timestamp	query		string	true	"RFC3339 timestamp to query positions at (e.g. 2026-04-25T15:20:00+00:00)"
//	@Success	200			{array}		slotPositionAtTime
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/events/{id}/slot-positions [get]
func GetSlotPositions(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	timestampStr := c.Query("timestamp")
	if timestampStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "timestamp query parameter required")
	}

	requestedTime, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid timestamp format, use RFC3339")
	}

	var event models.VATSIMEvent
	if database.DB.First(&event, eventID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	var revision models.SlotRevision
	if err := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport").
		Preload("Slots.ArrivalAirport").
		Where("event_id = ? AND EXISTS (SELECT 1 FROM slots WHERE slot_revision_id = slot_revisions.id)", eventID).
		Order("number DESC").
		First(&revision).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "no slot revision found for this event")
	}

	if len(revision.Slots) == 0 {
		return c.JSON([]slotPositionAtTime{})
	}

	slotIDs := make([]uint, len(revision.Slots))
	slotMap := make(map[uint]*models.Slot, len(revision.Slots))
	for i := range revision.Slots {
		slotIDs[i] = revision.Slots[i].ID
		slotMap[revision.Slots[i].ID] = &revision.Slots[i]
	}

	type positionResult struct {
		SlotID    uint
		Latitude  float64
		Longitude float64
		TimeDiff  time.Duration
	}

	closestBySlot := make(map[uint]positionResult)

	var positions []models.SlotPosition
	database.DB.Where("slot_id IN ?", slotIDs).Find(&positions)

	for _, pos := range positions {
		if pos.Timestamp.Equal(requestedTime) {
			closestBySlot[pos.SlotID] = positionResult{
				SlotID:    pos.SlotID,
				Latitude:  pos.Latitude,
				Longitude: pos.Longitude,
				TimeDiff:  0,
			}
		}
	}

	for _, pos := range positions {
		slot := slotMap[pos.SlotID]
		if slot == nil {
			continue
		}
		if slot.DepartureTime.After(requestedTime) || slot.ProjectedArrivalTime.Before(requestedTime) {
			continue
		}
		diff := pos.Timestamp.Sub(requestedTime)
		if diff < 0 {
			diff = -diff
		}
		existing, ok := closestBySlot[pos.SlotID]
		if !ok || diff < existing.TimeDiff {
			closestBySlot[pos.SlotID] = positionResult{
				SlotID:    pos.SlotID,
				Latitude:  pos.Latitude,
				Longitude: pos.Longitude,
				TimeDiff:  diff,
			}
		}
	}

	sort.Slice(revision.Slots, func(i, j int) bool {
		return revision.Slots[i].DepartureTime.Before(revision.Slots[j].DepartureTime)
	})

	var result []slotPositionAtTime
	for _, slot := range revision.Slots {
		if slot.DepartureTime.After(requestedTime) || slot.ProjectedArrivalTime.Before(requestedTime) {
			continue
		}
		pos, ok := closestBySlot[slot.ID]
		if !ok {
			continue
		}
		result = append(result, slotPositionAtTime{
			SlotID:           slot.ID,
			DepartureTime:    slot.DepartureTime.UTC().Format(time.RFC3339),
			ArrivalTime:      slot.ProjectedArrivalTime.UTC().Format(time.RFC3339),
			DepartureAirport: slot.DepartureAirport.Waypoint.Identifier,
			ArrivalAirport:   slot.ArrivalAirport.Waypoint.Identifier,
			Latitude:         pos.Latitude,
			Longitude:        pos.Longitude,
		})
	}

	return c.JSON(result)
}
