package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm/clause"
)

type highlightedWaypointInput struct {
	Identifier string  `json:"identifier"`
	Color      string  `json:"color"`
	Note       string  `json:"note"`
	WaypointID *int64  `json:"waypointId"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

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
	var input highlightedWaypointInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	waypointID := input.WaypointID
	lat, lon := input.Latitude, input.Longitude

	var existing models.Waypoint
	if database.DB.Where("identifier = ?", input.Identifier).First(&existing).Error == nil {
		if waypointID == nil {
			waypointID = &existing.ID
		}
		if lat == 0 {
			lat = existing.Latitude
		}
		if lon == 0 {
			lon = existing.Longitude
		}
	}

	if waypointID != nil {
		database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"identifier", "latitude", "longitude"}),
		}).Create(&models.Waypoint{
			ID:         *waypointID,
			Identifier: input.Identifier,
			Latitude:   lat,
			Longitude:  lon,
		})
	}

	wp := models.HighlightedWaypoint{
		Identifier: input.Identifier,
		Color:      input.Color,
		Note:       input.Note,
		WaypointID: waypointID,
	}

	result := database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identifier"}},
		DoUpdates: clause.AssignmentColumns([]string{"color", "note", "waypoint_id"}),
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
