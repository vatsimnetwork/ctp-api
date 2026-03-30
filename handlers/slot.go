package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

// ListSlotRevisions godoc
//
//	@Summary	List slot revisions for an event
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Success	200		{array}		models.SlotRevision
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/slot-revisions [get]
func ListSlotRevisions(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var revisions []models.SlotRevision
	if err := database.DB.
		Where("event_id = ?", eventID).
		Order("number DESC").
		Find(&revisions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(revisions)
}

// GetLatestSlotRevision godoc
//
//	@Summary	Get latest slot revision for an event (fully preloaded)
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Success	200		{object}	models.SlotRevision
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/slot-revisions/latest [get]
func GetLatestSlotRevision(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var revision models.SlotRevision
	result := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport").
		Preload("Slots.ArrivalAirport").
		Preload("Slots.RouteSegments").
		Preload("ThroughputStates").
		Where("event_id = ?", eventID).
		Order("number DESC").
		First(&revision)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "no slot revisions found for this event")
	}

	return c.JSON(revision)
}

// GetSlotRevision godoc
//
//	@Summary	Get slot revision by number
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Param		number	path		int	true	"Revision number"
//	@Success	200		{object}	models.SlotRevision
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/slot-revisions/{number} [get]
func GetSlotRevision(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}
	number, err := strconv.ParseUint(c.Params("number"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision number")
	}

	var revision models.SlotRevision
	result := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport").
		Preload("Slots.ArrivalAirport").
		Preload("Slots.RouteSegments").
		Preload("ThroughputStates").
		Where("event_id = ? AND number = ?", eventID, number).
		First(&revision)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	return c.JSON(revision)
}

// CreateSlotRevision godoc
//
//	@Summary	Create slot revision
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		eventId		path		int					true	"Event ID"
//	@Param		revision	body		models.SlotRevision	true	"Slot revision (number auto-assigned if 0)"
//	@Success	201			{object}	models.SlotRevision
//	@Failure	400			{object}	models.ErrorResponse
//	@Router		/events/{eventId}/slot-revisions [post]
func CreateSlotRevision(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var revision models.SlotRevision
	if err := c.Bind().JSON(&revision); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	revision.EventID = uint(eventID)

	if revision.Number == 0 {
		var maxNumber uint
		database.DB.Model(&models.SlotRevision{}).
			Where("event_id = ?", eventID).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNumber)
		revision.Number = maxNumber + 1
	}

	if err := database.DB.Create(&revision).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(revision)
}

// AddSlotsToRevision godoc
//
//	@Summary	Add slots to a slot revision
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		revisionId	path		int				true	"Slot revision ID"
//	@Param		slots		body		[]models.Slot	true	"Slots to add"
//	@Success	201			{array}		models.Slot
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/slots [post]
func AddSlotsToRevision(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var slots []models.Slot
	if err := c.Bind().JSON(&slots); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for i := range slots {
			// Extract route segment IDs before clearing the association to prevent
			// GORM from attempting to upsert existing route segment records on Create.
			rsIDs := make([]uint, 0, len(slots[i].RouteSegments))
			for _, rs := range slots[i].RouteSegments {
				rsIDs = append(rsIDs, rs.ID)
			}
			slots[i].RouteSegments = nil
			slots[i].SlotRevisionID = uint(revisionID)

			if err := tx.Create(&slots[i]).Error; err != nil {
				return err
			}

			if len(rsIDs) > 0 {
				rsegs := make([]models.RouteSegment, 0, len(rsIDs))
				for _, id := range rsIDs {
					rsegs = append(rsegs, models.RouteSegment{ThroughputPoint: models.ThroughputPoint{ID: id}})
				}
				if err := tx.Model(&slots[i]).Association("RouteSegments").Append(rsegs); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(slots)
}

// AddThroughputStatesToRevision godoc
//
//	@Summary	Add throughput states to a slot revision
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		revisionId	path		int							true	"Slot revision ID"
//	@Param		states		body		[]models.ThroughputState	true	"Throughput states"
//	@Success	201			{array}		models.ThroughputState
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/throughput-states [post]
func AddThroughputStatesToRevision(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var states []models.ThroughputState
	if err := c.Bind().JSON(&states); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	for i := range states {
		states[i].SlotRevisionID = uint(revisionID)
	}

	if err := database.DB.Create(&states).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(states)
}

// AddThroughputSnapshotsToRevision godoc
//
//	@Summary	Add throughput snapshots to a slot revision
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		revisionId	path		int								true	"Slot revision ID"
//	@Param		snapshots	body		[]models.ThroughputSnapshot		true	"Throughput snapshots"
//	@Success	201			{array}		models.ThroughputSnapshot
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/throughput-snapshots [post]
func AddThroughputSnapshotsToRevision(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var snapshots []models.ThroughputSnapshot
	if err := c.Bind().JSON(&snapshots); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	for i := range snapshots {
		snapshots[i].SlotRevisionID = uint(revisionID)
	}

	if err := database.DB.Create(&snapshots).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(snapshots)
}

// UpdateSlotRevision godoc
//
//	@Summary	Update slot revision metadata
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		revisionId	path		int					true	"Slot revision ID"
//	@Param		revision	body		models.SlotRevision	true	"Fields to update"
//	@Success	200			{object}	models.SlotRevision
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId} [put]
func UpdateSlotRevision(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var updates models.SlotRevision
	if err := c.Bind().JSON(&updates); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	updates.ID = uint(revisionID)
	if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.First(&existing, revisionID)
	return c.JSON(existing)
}

// DeleteSlotRevision godoc
//
//	@Summary	Delete slot revision
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		revisionId	path		int			true	"Slot revision ID"
//	@Success	200			{object}	models.SuccessResponse
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId} [delete]
func DeleteSlotRevision(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		return tx.Delete(&models.SlotRevision{}, revisionID).Error
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}
