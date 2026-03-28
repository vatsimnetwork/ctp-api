package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ListAirways godoc
//
//	@Summary	List all airways with waypoints
//	@Tags		airways
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.Airway
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/airways [get]
func ListAirways(c fiber.Ctx) error {
	var airways []models.Airway
	if err := database.DB.Preload("Waypoints").Preload("Waypoints.Location").Find(&airways).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(airways)
}

// CreateAirway godoc
//
//	@Summary	Create airway
//	@Tags		airways
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		airway	body		models.Airway	true	"Airway"
//	@Success	201		{object}	models.Airway
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/airways [post]
func CreateAirway(c fiber.Ctx) error {
	var airway models.Airway
	if err := c.Bind().JSON(&airway); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := database.DB.Create(&airway).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(airway)
}

// BulkCreateAirways godoc
//
//	@Summary	Bulk create airways
//	@Tags		airways
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		airways	body		[]models.Airway	true	"Airways"
//	@Success	201		{array}		models.Airway
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/airways/bulk [post]
func BulkCreateAirways(c fiber.Ctx) error {
	var airways []models.Airway
	if err := c.Bind().JSON(&airways); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := database.DB.Create(&airways).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(airways)
}

// DeleteAirway godoc
//
//	@Summary	Delete airway by identifier
//	@Tags		airways
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		identifier	path		string		true	"Airway identifier"
//	@Success	200			{object}	models.SuccessResponse
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/airways/{identifier} [delete]
func DeleteAirway(c fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "identifier required")
	}

	result := database.DB.Where("identifier = ?", identifier).Delete(&models.Airway{})
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "airway not found")
	}

	return c.JSON(fiber.Map{"success": true})
}

// DeleteAllAirways godoc
//
//	@Summary	Delete all airways
//	@Tags		airways
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/airways [delete]
func DeleteAllAirways(c fiber.Ctx) error {
	if err := database.DB.Where("1 = 1").Delete(&models.Airway{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"success": true})
}
