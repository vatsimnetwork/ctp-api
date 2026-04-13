package handlers

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

var selcalLetters = []string{
	"A", "B", "C", "D", "E", "F", "G", "H",
	"J", "K", "L", "M", "P", "Q", "R", "S",
}

// generateSelcal returns a unique SELCAL code in the form "AB-CD" where the
// four letters are distinct, each pair is sorted, and the code does not
// already appear in used. The caller is responsible for marking it used.
func generateSelcal(used map[string]struct{}, rng *rand.Rand) string {
	for {
		perm := rng.Perm(len(selcalLetters))[:4]
		a, b := selcalLetters[perm[0]], selcalLetters[perm[1]]
		c, d := selcalLetters[perm[2]], selcalLetters[perm[3]]
		if a > b {
			a, b = b, a
		}
		if c > d {
			c, d = d, c
		}
		code := a + b + "-" + c + d
		if _, exists := used[code]; exists {
			continue
		}
		used[code] = struct{}{}
		return code
	}
}

// ExportLatestSlotRevisionCSV godoc
//
//	@Summary	Export latest slot revision as CSV
//	@Description	Returns a CSV with columns: id,departure,arrival,tot,is_domestic,track,route,selcal
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
		Preload("Slots.RouteSegments").
		Where("event_id = ?", eventID).
		Order("number DESC").
		First(&revision)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "no slot revisions found for this event")
	}

	sortSlotRouteSegments(revision.Slots)

	used := make(map[string]struct{}, len(revision.Slots))
	rng := rand.New(rand.NewSource(int64(revision.ID)))

	var b strings.Builder
	b.WriteString("id,departure,arrival,tot,is_domestic,track,route,level,selcal\n")

	for _, s := range revision.Slots {
		dep := s.DepartureAirport.Waypoint.Identifier
		arr := s.ArrivalAirport.Waypoint.Identifier
		tot := s.DepartureTime.UTC().Format("2006-01-02 15:04:05")

		var track string
		routeParts := make([]string, 0, len(s.RouteSegments))
		for _, rs := range s.RouteSegments {
			if rs.RouteSegmentGroup == "OCA" && track == "" {
				track = rs.Identifier
			}
			if rs.RouteString != "" {
				routeParts = append(routeParts, rs.RouteString)
			}
		}
		route := dedupeConsecutive(strings.Join(routeParts, " "))
		selcal := generateSelcal(used, rng)

		fmt.Fprintf(&b, "%d,%s,%s,%s,false,%s,%s,,%s\n",
			s.ID, dep, arr, tot, track, csvEscape(route), selcal)
	}

	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition,
		fmt.Sprintf(`attachment; filename="slots-event%d-rev%d.csv"`, eventID, revision.Number))
	return c.SendString(b.String())
}

func dedupeConsecutive(route string) string {
	tokens := strings.Fields(route)
	out := tokens[:0:0]
	for _, t := range tokens {
		if len(out) == 0 || out[len(out)-1] != t {
			out = append(out, t)
		}
	}
	return strings.Join(out, " ")
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
