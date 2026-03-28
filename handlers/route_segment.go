package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

// ListAllRouteSegments godoc
//
//	@Summary	List all route segments
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.RouteSegment
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/route-segments [get]
func ListAllRouteSegments(c fiber.Ctx) error {
	var segments []models.RouteSegment
	if err := database.DB.Preload("Tags").Preload("Locations").Find(&segments).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(segments)
}

// ListEventRouteSegments godoc
//
//	@Summary	List route segments for an event
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Success	200		{array}		models.RouteSegment
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/route-segments [get]
func ListEventRouteSegments(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var segments []models.RouteSegment
	if err := database.DB.
		Preload("Tags").
		Preload("Locations").
		Preload("ProvidedFacilityProgression").
		Where("event_id = ?", eventID).
		Find(&segments).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(segments)
}

// CreateRouteSegment godoc
//
//	@Summary	Create route segment
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		segment	body		models.RouteSegment	true	"Route segment"
//	@Success	201		{object}	models.RouteSegment
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/route-segments [post]
func CreateRouteSegment(c fiber.Ctx) error {
	var segment models.RouteSegment
	if err := c.Bind().JSON(&segment); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := database.DB.Create(&segment).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(segment)
}

// BatchSaveRouteSegments godoc
//
//	@Summary	Batch create/update/delete route segments in a single transaction
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		models.BatchSaveRequest	true	"Batch payload: updates (array of RouteSegment) and deletes (array of IDs)"
//	@Success	200		{object}	models.SuccessResponse									"success: true"
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/route-segments/save [post]
func BatchSaveRouteSegments(c fiber.Ctx) error {
	var payload struct {
		Updates []models.RouteSegment `json:"updates"`
		Deletes []uint                `json:"deletes"`
	}
	if err := c.Bind().JSON(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, del := range payload.Deletes {
			if err := tx.Delete(&models.RouteSegment{}, del).Error; err != nil {
				return err
			}
		}

		for i := range payload.Updates {
			seg := &payload.Updates[i]
			if seg.ID == 0 {
				if err := tx.Create(seg).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(seg).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}

// UpdateRouteSegment godoc
//
//	@Summary	Update route segment
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int					true	"Route segment ID"
//	@Param		segment	body		models.RouteSegment	true	"Route segment fields to update"
//	@Success	200		{object}	models.RouteSegment
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/route-segments/{id} [put]
func UpdateRouteSegment(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid route segment id")
	}

	var existing models.RouteSegment
	if database.DB.First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "route segment not found")
	}

	var updates models.RouteSegment
	if err := c.Bind().JSON(&updates); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	updates.ID = uint(id)
	if err := database.DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.Preload("Tags").Preload("Locations").Preload("ProvidedFacilityProgression").First(&existing, id)
	return c.JSON(existing)
}

// DeleteRouteSegment godoc
//
//	@Summary	Delete route segment
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int			true	"Route segment ID"
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/route-segments/{id} [delete]
func DeleteRouteSegment(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid route segment id")
	}

	result := database.DB.Delete(&models.RouteSegment{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "route segment not found")
	}

	return c.JSON(fiber.Map{"success": true})
}
