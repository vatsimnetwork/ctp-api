package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ListAirports godoc
//
//	@Summary	List airports for an event
//	@Tags		airports
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Success	200		{array}		models.Airport
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/airports [get]
func ListAirports(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var airports []models.Airport
	if err := database.DB.Where("event_id = ?", eventID).Find(&airports).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(airports)
}

// CreateAirport godoc
//
//	@Summary	Create airport
//	@Tags		airports
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		eventId	path		int				true	"Event ID"
//	@Param		airport	body		models.Airport	true	"Airport"
//	@Success	201		{object}	models.Airport
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/airports [post]
func CreateAirport(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var airport models.Airport
	if err := c.Bind().JSON(&airport); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	airport.EventID = uint(eventID)
	if err := database.DB.Create(&airport).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(airport)
}

// UpdateAirport godoc
//
//	@Summary	Update airport
//	@Tags		airports
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int				true	"Airport ID"
//	@Param		airport	body		models.Airport	true	"Airport fields to update"
//	@Success	200		{object}	models.Airport
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/airports/{id} [put]
func UpdateAirport(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid airport id")
	}

	var existing models.Airport
	if database.DB.First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "airport not found")
	}

	var updates models.Airport
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

// DeleteAirport godoc
//
//	@Summary	Delete airport
//	@Tags		airports
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int			true	"Airport ID"
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/airports/{id} [delete]
func DeleteAirport(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid airport id")
	}

	result := database.DB.Delete(&models.Airport{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "airport not found")
	}

	return c.JSON(fiber.Map{"success": true})
}
