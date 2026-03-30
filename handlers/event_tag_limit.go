package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

type tagLimitResponse struct {
	Tag                    string `json:"tag"`
	MaximumAircraftPerHour uint16 `json:"maximumAircraftPerHour"`
}

func ListEventTagLimits(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var tags []models.EventTag
	if err := database.DB.Where("event_id = ?", eventID).Order("name ASC").Find(&tags).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]tagLimitResponse, 0, len(tags))
	for _, t := range tags {
		var limit uint16
		if t.MaximumAircraftPerHour != nil {
			limit = *t.MaximumAircraftPerHour
		}
		result = append(result, tagLimitResponse{Tag: t.Name, MaximumAircraftPerHour: limit})
	}
	return c.JSON(result)
}

func UpsertEventTagLimits(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var input []struct {
		Tag                    string `json:"tag"`
		MaximumAircraftPerHour uint16 `json:"maximumAircraftPerHour"`
	}
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	for _, item := range input {
		v := item.MaximumAircraftPerHour
		if err := database.DB.Model(&models.EventTag{}).
			Where("event_id = ? AND name = ?", eventID, item.Tag).
			UpdateColumn("maximum_aircraft_per_hour", v).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	return ListEventTagLimits(c)
}
