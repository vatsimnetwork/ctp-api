package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm/clause"
)

// ListHighlightedWaypoints godoc
//
//	@Summary	List all highlighted waypoints
//	@Tags		highlighted-waypoints
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.HighlightedWaypoint
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/highlighted-waypoints [get]
func ListHighlightedWaypoints(c fiber.Ctx) error {
	var waypoints []models.HighlightedWaypoint
	if err := database.DB.Order("identifier").Find(&waypoints).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(waypoints)
}

// UpsertHighlightedWaypoint godoc
//
//	@Summary	Create or update highlighted waypoint (upsert by identifier)
//	@Tags		highlighted-waypoints
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		waypoint	body		models.HighlightedWaypoint	true	"Highlighted waypoint"
//	@Success	201			{object}	models.HighlightedWaypoint
//	@Failure	400			{object}	models.ErrorResponse
//	@Router		/highlighted-waypoints [post]
func UpsertHighlightedWaypoint(c fiber.Ctx) error {
	var wp models.HighlightedWaypoint
	if err := c.Bind().JSON(&wp); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	result := database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identifier"}},
		DoUpdates: clause.AssignmentColumns([]string{"color", "note"}),
	}).Create(&wp)

	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(wp)
}

// DeleteHighlightedWaypoint godoc
//
//	@Summary	Delete highlighted waypoint
//	@Tags		highlighted-waypoints
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		identifier	path		string		true	"Waypoint identifier"
//	@Success	200			{object}	models.SuccessResponse
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/highlighted-waypoints/{identifier} [delete]
func DeleteHighlightedWaypoint(c fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "identifier required")
	}

	result := database.DB.Where("identifier = ?", identifier).Delete(&models.HighlightedWaypoint{})
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "highlighted waypoint not found")
	}

	return c.JSON(fiber.Map{"success": true})
}
