package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ListGeoJsonOverlays godoc
//
//	@Summary	List all GeoJSON overlays
//	@Tags		geo-json-overlays
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.GeoJsonOverlay
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/geo-json-overlays [get]
func ListGeoJsonOverlays(c fiber.Ctx) error {
	var overlays []models.GeoJsonOverlay
	if err := database.DB.Order("name").Find(&overlays).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(overlays)
}

// UpsertGeoJsonOverlay godoc
//
//	@Summary	Create or update a GeoJSON overlay
//	@Tags		geo-json-overlays
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		overlay	body		models.GeoJsonOverlay	true	"GeoJSON overlay"
//	@Success	201	{object}	models.GeoJsonOverlay
//	@Failure	400	{object}	models.ErrorResponse
//	@Router		/geo-json-overlays [post]
func UpsertGeoJsonOverlay(c fiber.Ctx) error {
	var overlay models.GeoJsonOverlay
	if err := c.Bind().JSON(&overlay); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if overlay.Name == "" || overlay.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and url are required")
	}

	if overlay.ID != 0 {
		result := database.DB.Save(&overlay)
		if result.Error != nil {
			return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
		}
		return c.Status(fiber.StatusOK).JSON(overlay)
	}

	result := database.DB.Create(&overlay)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(overlay)
}

// DeleteGeoJsonOverlay godoc
//
//	@Summary	Delete a GeoJSON overlay
//	@Tags		geo-json-overlays
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Overlay ID"
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/geo-json-overlays/{id} [delete]
func DeleteGeoJsonOverlay(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	result := database.DB.Delete(&models.GeoJsonOverlay{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "overlay not found")
	}

	return c.JSON(fiber.Map{"success": true})
}
