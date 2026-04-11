package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

func ListDeparturePairPreferences(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var prefs []models.DeparturePairPreference
	if err := database.DB.Where("event_id = ?", eventID).Find(&prefs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([][]interface{}, 0, len(prefs))
	for _, p := range prefs {
		result = append(result, []interface{}{p.DepartureAirportID, p.ArrivalAirportID, p.Preference})
	}
	return c.JSON(result)
}

func SetDeparturePairPreferences(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var input [][]interface{}
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	tx := database.DB.Begin()

	if err := tx.Where("event_id = ?", eventID).Delete(&models.DeparturePairPreference{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	for _, item := range input {
		if len(item) != 3 {
			continue
		}
		depID, ok1 := item[0].(float64)
		arrID, ok2 := item[1].(float64)
		pref, ok3 := item[2].(float64)
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		prefUint8 := uint8(pref)
		if prefUint8 == models.PreferenceNone {
			continue
		}
		p := models.DeparturePairPreference{
			EventID:            uint(eventID),
			DepartureAirportID: uint(depID),
			ArrivalAirportID:   uint(arrID),
			Preference:         prefUint8,
		}
		if err := tx.Create(&p).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(input)
}
