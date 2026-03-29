package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ListSectors godoc
//
//	@Summary	List sectors for an event
//	@Tags		sectors
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Success	200		{array}		models.Sector
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/sectors [get]
func ListSectors(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var sectors []models.Sector
	if err := database.DB.
		Preload("SectorBoundaries").
		Preload("SectorBoundaries.Coordinates").
		Where("event_id = ? OR event_id IS NULL", eventID).
		Find(&sectors).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(sectors)
}

// CreateSector godoc
//
//	@Summary	Create sector
//	@Tags		sectors
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		eventId	path		int				true	"Event ID"
//	@Param		sector	body		models.Sector	true	"Sector"
//	@Success	201		{object}	models.Sector
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/sectors [post]
func CreateSector(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var sector models.Sector
	if err := c.Bind().JSON(&sector); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	eid := uint(eventID)
	sector.EventID = &eid
	if err := database.DB.Create(&sector).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(sector)
}

// UpdateSector godoc
//
//	@Summary	Update sector
//	@Tags		sectors
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int				true	"Sector ID"
//	@Param		sector	body		models.Sector	true	"Sector fields to update"
//	@Success	200		{object}	models.Sector
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/sectors/{id} [put]
func UpdateSector(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sector id")
	}

	var existing models.Sector
	if database.DB.Preload("SectorBoundaries").
		Preload("SectorBoundaries.Coordinates").First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "sector not found")
	}

	var updates models.Sector
	if err := c.Bind().JSON(&updates); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	updates.ID = uint(id)
	if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.Preload("SectorBoundaries").
		Preload("SectorBoundaries.Coordinates").First(&existing, id)
	return c.JSON(existing)
}

// DeleteSector godoc
//
//	@Summary	Delete sector
//	@Tags		sectors
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int			true	"Sector ID"
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/sectors/{id} [delete]
func DeleteSector(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sector id")
	}

	result := database.DB.Delete(&models.Sector{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "sector not found")
	}

	return c.JSON(fiber.Map{"success": true})
}
