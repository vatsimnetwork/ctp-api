package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ListEvents godoc
//
//	@Summary	List all events
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.VATSIMEvent
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/events [get]
func ListEvents(c fiber.Ctx) error {
	var events []models.VATSIMEvent
	if err := database.DB.Find(&events).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(events)
}

// GetEvent godoc
//
//	@Summary	Get event with all nested data
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	models.VATSIMEvent
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id} [get]
func GetEvent(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var event models.VATSIMEvent
	result := database.DB.
		Preload("Airports").
		Preload("Airports.Waypoint").
		Preload("RouteSegments").
		Preload("RouteSegments.Tags").
		Preload("RouteSegments.Locations").
		Preload("RouteSegments.Locations.Waypoint").
		Preload("RouteSegments.ProvidedFacilityProgression").
		Preload("Sectors").
		Preload("Sectors.SectorBoundaries").
		Preload("Sectors.SectorBoundaries.Coordinates").
		Preload("SlotRevisions").
		First(&event, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	return c.JSON(event)
}

// CreateEvent godoc
//
//	@Summary	Create event
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		event	body		models.VATSIMEvent	true	"Event"
//	@Success	201		{object}	models.VATSIMEvent
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events [post]
func CreateEvent(c fiber.Ctx) error {
	var event models.VATSIMEvent
	if err := c.Bind().JSON(&event); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := database.DB.Create(&event).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(event)
}

// UpdateEvent godoc
//
//	@Summary	Update event
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int					true	"Event ID"
//	@Param		event	body		models.VATSIMEvent	true	"Event fields to update"
//	@Success	200		{object}	models.VATSIMEvent
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/events/{id} [put]
func UpdateEvent(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var existing models.VATSIMEvent
	if database.DB.First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	var updates models.VATSIMEvent
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

// PatchEventCalculationParams godoc
//
//	@Summary	Patch event calculation parameters
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int						true	"Event ID"
//	@Success	200	{object}	models.VATSIMEvent
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id}/calculation-params [patch]
func PatchEventCalculationParams(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var existing models.VATSIMEvent
	if database.DB.First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	// Bind to a map so zero-value booleans/ints from the JSON body are preserved.
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Only allow updating calculation parameter columns. Map camelCase JSON keys
	// to snake_case GORM column names so zero values (false, 0) are written correctly.
	allowed := map[string]string{
		"intendedSlotGenerationMode":                            "intended_slot_generation_mode",
		"departureTimeWindowOffsetSynchronizationLongitude":     "departure_time_window_offset_synchronization_longitude",
		"simulationAnalysisResolutionInMinutes":                 "simulation_analysis_resolution_in_minutes",
		"shouldSimulationUseActualWeatherForecastData":          "should_simulation_use_actual_weather_forecast_data",
		"intendedDepartureTimeWindowOffsetsCalculationMode":     "intended_departure_time_window_offsets_calculation_mode",
		"departureTimeWindowOffsetSynchronizationTimeOfDay":     "departure_time_window_offset_synchronization_time_of_day",
		"calculateThroughputDataOnlyForManuallyProvidedSectors": "calculate_throughput_data_only_for_manually_provided_sectors",
		"intendedWaypointThroughputCalculationMode":             "intended_waypoint_throughput_calculation_mode",
		"thresholdToCheckIfAirplaneIsCountedAtWaypointInNm":     "threshold_to_check_if_airplane_is_counted_at_waypoint_in_nm",
		"calculationFallbackGroundSpeed":                        "calculation_fallback_ground_speed",
		"highSimulationAccuracy":                                "high_simulation_accuracy",
	}

	updates := make(map[string]interface{}, len(body))
	for jsonKey, colName := range allowed {
		if val, ok := body[jsonKey]; ok {
			updates[colName] = val
		}
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	database.DB.First(&existing, id)
	return c.JSON(existing)
}

// DeleteEvent godoc
//
//	@Summary	Delete event
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int		true	"Event ID"
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id} [delete]
func DeleteEvent(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	result := database.DB.Delete(&models.VATSIMEvent{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	return c.JSON(fiber.Map{"success": true})
}

// GetSimulatorData godoc
//
//	@Summary	Get full simulator data for an event
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id			path		int		true	"Event ID"
//	@Param		revision	query		int		false	"Specific slot revision number (defaults to latest)"
//	@Success	200			{object}	models.SimulatorDataResponse
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/events/{id}/simulator-data [get]
func GetSimulatorData(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	revisionParam := c.Query("revision")

	var event models.VATSIMEvent
	result := database.DB.
		Preload("Airports").
		Preload("Airports.Waypoint").
		Preload("RouteSegments").
		Preload("RouteSegments.Tags").
		Preload("RouteSegments.Locations").
		Preload("RouteSegments.Locations.Waypoint").
		Preload("RouteSegments.ProvidedFacilityProgression").
		Preload("RouteSegments.ProvidedFacilityProgression.SectorBoundaries").
		Preload("RouteSegments.ProvidedFacilityProgression.SectorBoundaries.Coordinates").
		Preload("Sectors", "event_id IS NULL OR event_id = ?", id).
		Preload("Sectors.SectorBoundaries").
		Preload("Sectors.SectorBoundaries.Coordinates").
		First(&event, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	var revision models.SlotRevision
	query := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport").
		Preload("Slots.DepartureAirport.Waypoint").
		Preload("Slots.ArrivalAirport").
		Preload("Slots.ArrivalAirport.Waypoint").
		Preload("Slots.RouteSegments").
		Preload("Slots.RouteSegments.Tags").
		Preload("Slots.RouteSegments.Locations").
		Preload("Slots.RouteSegments.Locations.Waypoint").
		Preload("Slots.RouteSegments.ProvidedFacilityProgression").
		Preload("ThroughputStates").
		Preload("ThroughputSnapshots")

	if revisionParam != "" {
		num, err := strconv.ParseUint(revisionParam, 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid revision number")
		}
		query = query.Where("event_id = ? AND number = ?", id, num)
	} else {
		query = query.Where("event_id = ?", id).Order("number DESC")
	}

	if query.First(&revision).Error != nil {
		return c.JSON(fiber.Map{
			"event":    event,
			"revision": nil,
		})
	}

	return c.JSON(fiber.Map{
		"event":    event,
		"revision": revision,
	})
}

// GetSimulatorDataLatestWithSlots godoc
//
//	@Summary	Get simulator data for the latest revision that has slots
//	@Tags		events
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	models.SimulatorDataResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id}/simulator-data/latest-with-slots [get]
func GetSimulatorDataLatestWithSlots(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var event models.VATSIMEvent
	result := database.DB.
		Preload("Airports").
		Preload("Airports.Waypoint").
		Preload("RouteSegments").
		Preload("RouteSegments.Tags").
		Preload("RouteSegments.Locations").
		Preload("RouteSegments.Locations.Waypoint").
		Preload("RouteSegments.ProvidedFacilityProgression").
		Preload("RouteSegments.ProvidedFacilityProgression.SectorBoundaries").
		Preload("RouteSegments.ProvidedFacilityProgression.SectorBoundaries.Coordinates").
		Preload("Sectors", "event_id IS NULL OR event_id = ?", id).
		Preload("Sectors.SectorBoundaries").
		Preload("Sectors.SectorBoundaries.Coordinates").
		First(&event, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	var revision models.SlotRevision
	query := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport").
		Preload("Slots.DepartureAirport.Waypoint").
		Preload("Slots.ArrivalAirport").
		Preload("Slots.ArrivalAirport.Waypoint").
		Preload("Slots.RouteSegments").
		Preload("Slots.RouteSegments.Tags").
		Preload("Slots.RouteSegments.Locations").
		Preload("Slots.RouteSegments.Locations.Waypoint").
		Preload("Slots.RouteSegments.ProvidedFacilityProgression").
		Preload("ThroughputStates").
		Preload("ThroughputSnapshots").
		Where("event_id = ? AND EXISTS (SELECT 1 FROM slots WHERE slots.slot_revision_id = slot_revisions.id LIMIT 1)", id).
		Order("number DESC")

	if query.First(&revision).Error != nil {
		return c.JSON(fiber.Map{
			"event":    event,
			"revision": nil,
		})
	}

	return c.JSON(fiber.Map{
		"event":    event,
		"revision": revision,
	})
}
