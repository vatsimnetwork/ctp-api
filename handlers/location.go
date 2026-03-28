package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ListWaypoints godoc
//
//	@Summary	List waypoints for an event
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Success	200		{array}		models.Location
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/waypoints [get]
func ListWaypoints(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var locations []models.Location
	if err := database.DB.Where("event_id = ?", eventID).Find(&locations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(locations)
}

// CreateWaypoint godoc
//
//	@Summary	Create waypoint
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		eventId		path		int				true	"Event ID"
//	@Param		waypoint	body		models.Location	true	"Waypoint"
//	@Success	201			{object}	models.Location
//	@Failure	400			{object}	models.ErrorResponse
//	@Router		/events/{eventId}/waypoints [post]
func CreateWaypoint(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var location models.Location
	if err := c.Bind().JSON(&location); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	eid := uint(eventID)
	location.EventID = &eid
	if err := database.DB.Create(&location).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(location)
}

// BulkCreateWaypoints godoc
//
//	@Summary	Bulk create waypoints
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		eventId		path		int					true	"Event ID"
//	@Param		waypoints	body		[]models.Location	true	"Waypoints"
//	@Success	201			{array}		models.Location
//	@Failure	400			{object}	models.ErrorResponse
//	@Router		/events/{eventId}/waypoints/bulk [post]
func BulkCreateWaypoints(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var locations []models.Location
	if err := c.Bind().JSON(&locations); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	eid := uint(eventID)
	for i := range locations {
		locations[i].EventID = &eid
	}

	if err := database.DB.Create(&locations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(locations)
}

// UpdateWaypoint godoc
//
//	@Summary	Update waypoint
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id			path		int				true	"Waypoint ID"
//	@Param		waypoint	body		models.Location	true	"Waypoint fields to update"
//	@Success	200			{object}	models.Location
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/waypoints/{id} [put]
func UpdateWaypoint(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid waypoint id")
	}

	var existing models.Location
	if database.DB.First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "waypoint not found")
	}

	var updates models.Location
	if err := c.Bind().JSON(&updates); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	updates.ID = uint(id)
	if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.First(&existing, id)
	return c.JSON(existing)
}

// DeleteWaypoint godoc
//
//	@Summary	Delete waypoint
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int			true	"Waypoint ID"
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/waypoints/{id} [delete]
func DeleteWaypoint(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid waypoint id")
	}

	result := database.DB.Delete(&models.Location{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "waypoint not found")
	}

	return c.JSON(fiber.Map{"success": true})
}

// DeleteAllWaypoints godoc
//
//	@Summary	Delete all waypoints for an event
//	@Tags		waypoints
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int			true	"Event ID"
//	@Success	200		{object}	models.SuccessResponse
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/waypoints [delete]
func DeleteAllWaypoints(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	if err := database.DB.Where("event_id = ?", eventID).Delete(&models.Location{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}
