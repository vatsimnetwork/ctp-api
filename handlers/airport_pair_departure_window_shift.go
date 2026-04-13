package handlers

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

// ListWindowShifts godoc
//
//	@Summary	List airport pair departure window shifts for a revision
//	@Tags		window-shifts
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		revisionId	path		int	true	"Slot revision ID"
//	@Success	200		{array}		models.AirportPairDepartureWindowShift
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/window-shifts [get]
func ListWindowShifts(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var shifts []models.AirportPairDepartureWindowShift
	if err := database.DB.
		Preload("DepartureAirport").
		Preload("DepartureAirport.Waypoint").
		Preload("ArrivalAirport").
		Preload("ArrivalAirport.Waypoint").
		Where("slot_revision_id = ?", revisionID).
		Find(&shifts).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(shifts)
}

// ReplaceWindowShifts godoc
//
//	@Summary	Replace all airport pair departure window shifts for a revision
//	@Tags		window-shifts
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		revisionId	path		int	true	"Slot revision ID"
//	@Param		shifts	body		[]WindowShiftInput	true	"Window shifts to set"
//	@Success	200		{object}	models.SuccessResponse
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/window-shifts [put]
func ReplaceWindowShifts(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var revision models.SlotRevision
	if database.DB.First(&revision, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var inputs []WindowShiftInput
	if err := c.Bind().JSON(&inputs); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Validate inputs
	for _, input := range inputs {
		if input.DepartureAirportID == 0 || input.ArrivalAirportID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "departureAirportId and arrivalAirportId are required")
		}
		if !isMultipleOfHalf(input.StartShiftHours) || !isMultipleOfHalf(input.EndShiftHours) {
			return fiber.NewError(fiber.StatusBadRequest, "shifts must be multiples of 0.5")
		}
		if input.StartShiftHours == 0 && input.EndShiftHours == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "omit rows where both shifts are 0")
		}
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("slot_revision_id = ?", revisionID).
			Delete(&models.AirportPairDepartureWindowShift{}).Error; err != nil {
			return err
		}
		for _, input := range inputs {
			shift := models.AirportPairDepartureWindowShift{
				SlotRevisionID:     uint(revisionID),
				DepartureAirportID: input.DepartureAirportID,
				ArrivalAirportID:   input.ArrivalAirportID,
				StartShiftHours:    input.StartShiftHours,
				EndShiftHours:      input.EndShiftHours,
			}
			if err := tx.Create(&shift).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}

func isMultipleOfHalf(v float64) bool {
	doubled := v * 2
	return math.Abs(doubled-math.Round(doubled)) < 1e-9
}
