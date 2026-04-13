package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

// ListSlotDraftEntries godoc
//
//	@Summary	List slot draft entries for a revision
//	@Tags		slot-draft-entries
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		revisionId	path		int	true	"Slot revision ID"
//	@Success	200		{array}		models.SlotDraftEntry
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/draft-entries [get]
func ListSlotDraftEntries(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var entries []models.SlotDraftEntry
	if err := database.DB.
		Preload("DepartureAirport").
		Preload("DepRoute").
		Preload("Track").
		Preload("ArrRoute").
		Preload("ArrivalAirport").
		Where("slot_revision_id = ?", revisionID).
		Find(&entries).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(entries)
}

// AddSlotDraftEntries godoc
//
//	@Summary	Add slot draft entries to a revision
//	@Tags		slot-draft-entries
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		revisionId	path		int	true	"Slot revision ID"
//	@Param		entries	body		[]SlotDraftEntryInput	true	"Slot draft entries to add"
//	@Success	201		{object}	models.SuccessResponse
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/draft-entries [post]
func AddSlotDraftEntries(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var inputs []SlotDraftEntryInput
	if err := c.Bind().JSON(&inputs); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for _, input := range inputs {
			entry := models.SlotDraftEntry{
				SlotRevisionID:     uint(revisionID),
				DepartureAirportID: input.DepartureAirportID,
				DepRouteID:         input.DepRouteID,
				TrackID:            input.TrackID,
				ArrRouteID:         input.ArrRouteID,
				ArrivalAirportID:   input.ArrivalAirportID,
				SlotCount:          input.SlotCount,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
}
