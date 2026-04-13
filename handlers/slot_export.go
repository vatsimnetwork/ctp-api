package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

// ExportLatestSlotRevisionCSV godoc
//
//	@Summary	Export latest slot revision as CSV
//	@Description	Returns a CSV with columns: departure,arrival,tot
//	@Tags		slot-revisions
//	@Security	ApiKeyAuth
//	@Produce	text/csv
//	@Param		eventId	path		int	true	"Event ID"
//	@Success	200		{string}	string
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/slot-revisions/latest/export [get]
func ExportLatestSlotRevisionCSV(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var revision models.SlotRevision
	result := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport.Waypoint").
		Preload("Slots.ArrivalAirport.Waypoint").
		Where("event_id = ?", eventID).
		Order("number DESC").
		First(&revision)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "no slot revisions found for this event")
	}

	var b strings.Builder
	b.WriteString("departure,arrival,tot\n")

	for _, s := range revision.Slots {
		dep := s.DepartureAirport.Waypoint.Identifier
		arr := s.ArrivalAirport.Waypoint.Identifier
		tot := s.DepartureTime.UTC().Format("15:04")

		fmt.Fprintf(&b, "%s,%s,%s\n", dep, arr, tot)
	}

	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition,
		fmt.Sprintf(`attachment; filename="slots-event%d-rev%d.csv"`, eventID, revision.Number))
	return c.SendString(b.String())
}
