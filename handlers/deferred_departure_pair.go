package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

func ListDeferredDeparturePairs(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var pairs []models.DeferredDeparturePair
	if err := database.DB.Where("event_id = ?", eventID).Find(&pairs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([][]uint, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, []uint{p.DepartureAirportID, p.ArrivalAirportID})
	}
	return c.JSON(result)
}

// SetDeferredDeparturePairs replaces all deferred pairs for the event.
// Input: [[departureAirportId, arrivalAirportId], ...]
func SetDeferredDeparturePairs(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var input [][]uint
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	tx := database.DB.Begin()

	// Delete existing pairs for this event
	if err := tx.Where("event_id = ?", eventID).Delete(&models.DeferredDeparturePair{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Insert new pairs
	for _, pair := range input {
		if len(pair) != 2 {
			continue
		}
		p := models.DeferredDeparturePair{
			EventID:            uint(eventID),
			DepartureAirportID: pair[0],
			ArrivalAirportID:   pair[1],
		}
		if err := tx.Create(&p).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(input)
}
