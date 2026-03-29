package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm/clause"
)

type airportInput struct {
	WaypointID             int64   `json:"waypointId"`
	Identifier             string  `json:"identifier"`
	Latitude               float64 `json:"latitude"`
	Longitude              float64 `json:"longitude"`
	MaximumAircraftPerHour uint16  `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16  `json:"maximumSlots"`
	NumberOfVotes          uint16  `json:"numberOfVotes"`
}

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
	if err := database.DB.Preload("Waypoint").Where("event_id = ?", eventID).Find(&airports).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(airports)
}

// CreateAirport godoc
//
//	@Summary	Create airport for an event
//	@Tags		airports
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		eventId	path		int				true	"Event ID"
//	@Param		airport	body		airportInput	true	"Airport"
//	@Success	201		{object}	models.Airport
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/airports [post]
func CreateAirport(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var input airportInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"identifier", "latitude", "longitude"}),
	}).Create(&models.Waypoint{
		ID:         input.WaypointID,
		Identifier: input.Identifier,
		Latitude:   input.Latitude,
		Longitude:  input.Longitude,
	}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	airport := models.Airport{
		WaypointID:             input.WaypointID,
		EventID:                uint(eventID),
		MaximumAircraftPerHour: input.MaximumAircraftPerHour,
		MaximumSlots:           input.MaximumSlots,
		NumberOfVotes:          input.NumberOfVotes,
	}
	if err := database.DB.Create(&airport).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.Preload("Waypoint").First(&airport, airport.ID)
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
//	@Param		airport	body		airportInput	true	"Airport fields to update"
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

	var input airportInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if input.WaypointID != 0 {
		if err := database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"identifier", "latitude", "longitude"}),
		}).Create(&models.Waypoint{
			ID:         input.WaypointID,
			Identifier: input.Identifier,
			Latitude:   input.Latitude,
			Longitude:  input.Longitude,
		}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	if err := database.DB.Model(&existing).Updates(map[string]any{
		"waypoint_id":               input.WaypointID,
		"maximum_aircraft_per_hour": input.MaximumAircraftPerHour,
		"maximum_slots":             input.MaximumSlots,
		"number_of_votes":           input.NumberOfVotes,
	}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.Preload("Waypoint").First(&existing, id)
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
