package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ListWaypoints godoc
//
//	@Summary	List all canonical waypoints
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.Waypoint
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/waypoints [get]
func ListWaypoints(c fiber.Ctx) error {
	var waypoints []models.Waypoint
	if err := database.DB.Find(&waypoints).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(waypoints)
}

// UpdateWaypoint godoc
//
//	@Summary	Update waypoint throughput settings
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int				true	"Waypoint ID (navdata ID)"
//	@Param		waypoint	body		models.Waypoint	true	"Fields to update"
//	@Success	200		{object}	models.Waypoint
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/waypoints/{id} [put]
func UpdateWaypoint(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid waypoint id")
	}

	var existing models.Waypoint
	if database.DB.First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "waypoint not found")
	}

	var updates struct {
		MaximumAircraftPerHour uint16 `json:"maximumAircraftPerHour"`
		MaximumSlots           uint16 `json:"maximumSlots"`
	}
	if err := c.Bind().JSON(&updates); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := database.DB.Model(&existing).Updates(map[string]any{
		"maximum_aircraft_per_hour": updates.MaximumAircraftPerHour,
		"maximum_slots":             updates.MaximumSlots,
	}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.First(&existing, id)
	return c.JSON(existing)
}
