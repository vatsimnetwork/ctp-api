package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm/clause"
)

// ListCustomFixes godoc
//
//	@Summary	List all custom fixes
//	@Tags		custom-fixes
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.CustomFix
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/custom-fixes [get]
func ListCustomFixes(c fiber.Ctx) error {
	var fixes []models.CustomFix
	if err := database.DB.Order("identifier").Find(&fixes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fixes)
}

// UpsertCustomFix godoc
//
//	@Summary	Create or update custom fix (upsert by identifier)
//	@Tags		custom-fixes
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		fix	body		models.CustomFix	true	"Custom fix"
//	@Success	201	{object}	models.CustomFix
//	@Failure	400	{object}	models.ErrorResponse
//	@Router		/custom-fixes [post]
func UpsertCustomFix(c fiber.Ctx) error {
	var fix models.CustomFix
	if err := c.Bind().JSON(&fix); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	result := database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identifier"}},
		DoUpdates: clause.AssignmentColumns([]string{"latitude", "longitude", "note"}),
	}).Create(&fix)

	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fix)
}

// DeleteCustomFix godoc
//
//	@Summary	Delete custom fix
//	@Tags		custom-fixes
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		identifier	path		string		true	"Custom fix identifier"
//	@Success	200			{object}	models.SuccessResponse
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Router		/custom-fixes/{identifier} [delete]
func DeleteCustomFix(c fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "identifier required")
	}

	result := database.DB.Where("identifier = ?", identifier).Delete(&models.CustomFix{})
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "custom fix not found")
	}

	return c.JSON(fiber.Map{"success": true})
}
