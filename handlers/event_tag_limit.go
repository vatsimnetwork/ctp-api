package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

type tagLimitResponse struct {
	Tag                    string  `json:"tag"`
	MaximumAircraftPerHour uint16  `json:"maximumAircraftPerHour"`
}

func ListEventTagLimits(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var tags []models.RouteSegmentTag
	if err := database.DB.
		Joins("JOIN route_segments ON route_segments.id = route_segment_tags.route_segment_id").
		Where("route_segments.event_id = ?", eventID).
		Find(&tags).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	seen := make(map[string]bool)
	limitByTag := make(map[string]*uint16)
	order := []string{}

	for _, t := range tags {
		if !seen[t.Tag] {
			seen[t.Tag] = true
			order = append(order, t.Tag)
		}
		if t.MaximumAircraftPerHour != nil && limitByTag[t.Tag] == nil {
			v := *t.MaximumAircraftPerHour
			limitByTag[t.Tag] = &v
		}
	}

	result := make([]tagLimitResponse, 0, len(order))
	for _, tag := range order {
		var limit uint16
		if limitByTag[tag] != nil {
			limit = *limitByTag[tag]
		}
		result = append(result, tagLimitResponse{Tag: tag, MaximumAircraftPerHour: limit})
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
		if err := database.DB.Model(&models.RouteSegmentTag{}).
			Where("tag = ? AND route_segment_id IN (SELECT id FROM route_segments WHERE event_id = ?)", item.Tag, eventID).
			UpdateColumn("maximum_aircraft_per_hour", v).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	return ListEventTagLimits(c)
}
