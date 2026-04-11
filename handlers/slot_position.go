package handlers

import (
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

func getSlotRevisionWithSlots(eventID uint64) (*models.SlotRevision, error) {
	var revision models.SlotRevision
	if err := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport", "id IS NOT NULL").
		Preload("Slots.DepartureAirport.Waypoint").
		Preload("Slots.ArrivalAirport", "id IS NOT NULL").
		Preload("Slots.ArrivalAirport.Waypoint").
		Where("event_id = ? AND EXISTS (SELECT 1 FROM slots WHERE slot_revision_id = slot_revisions.id)", eventID).
		Order("number DESC").
		First(&revision).Error; err != nil {
		return nil, err
	}
	return &revision, nil
}

// GetSlotPositionsAtTime godoc
//
//	@Summary	Get slot positions nearest to a specific timestamp for all slots in flight at that time
//	@Tags		slots
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id			path		int		true	"Event ID"
//	@Param		timestamp	query		string	true	"RFC3339 timestamp (e.g. 2026-04-25T15:20:00+00:00)"
//	@Success	200			{array}		slotPositionAtTime
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/events/{id}/slot-positions [get]
func GetSlotPositionsAtTime(c fiber.Ctx) error {
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

	revision, err := getSlotRevisionWithSlots(eventID)
	if err != nil {
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

	var positions []models.SlotPosition
	database.DB.Where("slot_id IN ?", slotIDs).Find(&positions)

	type positionResult struct {
		Latitude  float64
		Longitude float64
		TimeDiff  time.Duration
	}

	closestBySlot := make(map[uint]positionResult)

	for _, pos := range positions {
		if pos.Timestamp.Equal(requestedTime) {
			closestBySlot[pos.SlotID] = positionResult{
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
			SlotID:             slot.ID,
			DepartureTime:      slot.DepartureTime.UTC().Format(time.RFC3339),
			ArrivalTime:        slot.ProjectedArrivalTime.UTC().Format(time.RFC3339),
			DepartureAirport:   slot.DepartureAirport.Waypoint.Identifier,
			DepartureAirportID: slot.DepartureAirport.WaypointID,
			ArrivalAirport:     slot.ArrivalAirport.Waypoint.Identifier,
			ArrivalAirportID:   slot.ArrivalAirport.WaypointID,
			Latitude:           pos.Latitude,
			Longitude:          pos.Longitude,
		})
	}

	return c.JSON(result)
}

// GetAllSlotPositions godoc
//
//	@Summary	Get all slot positions for all slots
//	@Tags		slots
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{array}		allSlotPositions
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id}/slot-positions/all [get]
func GetAllSlotPositions(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var event models.VATSIMEvent
	if database.DB.First(&event, eventID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	revision, err := getSlotRevisionWithSlots(eventID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "no slot revision found for this event")
	}

	if len(revision.Slots) == 0 {
		return c.JSON([]allSlotPositions{})
	}

	slotIDs := make([]uint, len(revision.Slots))
	for i := range revision.Slots {
		slotIDs[i] = revision.Slots[i].ID
	}

	var positions []models.SlotPosition
	database.DB.Where("slot_id IN ?", slotIDs).Find(&positions)

	positionsBySlot := make(map[uint][]slotPositionTimestamp)
	for _, pos := range positions {
		positionsBySlot[pos.SlotID] = append(positionsBySlot[pos.SlotID], slotPositionTimestamp{
			Timestamp: pos.Timestamp.UTC().Format(time.RFC3339),
			Latitude:  pos.Latitude,
			Longitude: pos.Longitude,
		})
	}

	sort.Slice(revision.Slots, func(i, j int) bool {
		return revision.Slots[i].DepartureTime.Before(revision.Slots[j].DepartureTime)
	})

	var result []allSlotPositions
	for _, slot := range revision.Slots {
		posList := positionsBySlot[slot.ID]
		sort.Slice(posList, func(i, j int) bool {
			return posList[i].Timestamp < posList[j].Timestamp
		})
		result = append(result, allSlotPositions{
			SlotID:             slot.ID,
			DepartureTime:      slot.DepartureTime.UTC().Format(time.RFC3339),
			ArrivalTime:        slot.ProjectedArrivalTime.UTC().Format(time.RFC3339),
			DepartureAirport:   slot.DepartureAirport.Waypoint.Identifier,
			DepartureAirportID: slot.DepartureAirport.WaypointID,
			ArrivalAirport:     slot.ArrivalAirport.Waypoint.Identifier,
			ArrivalAirportID:   slot.ArrivalAirport.WaypointID,
			Positions:          posList,
		})
	}

	return c.JSON(result)
}
