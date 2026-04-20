package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

// ---- SELCAL helpers ----

// selcalLetters are the 16 valid SELCAL tone letters.
var selcalLetters = []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'J', 'K', 'L', 'M', 'P', 'Q', 'R', 'S'}

// allValidSelcals returns every valid SELCAL code as "AB-CD" strings.
// A valid code uses 4 distinct letters from selcalLetters, arranged as two
// alphabetically-ordered pairs. AB-CD and CD-AB are distinct codes.
func allValidSelcals() []string {
	n := len(selcalLetters)
	var codes []string
	for a := 0; a < n; a++ {
		for b := a + 1; b < n; b++ {
			for c := 0; c < n; c++ {
				if c == a || c == b {
					continue
				}
				for d := c + 1; d < n; d++ {
					if d == a || d == b {
						continue
					}
					codes = append(codes, fmt.Sprintf("%c%c-%c%c",
						selcalLetters[a], selcalLetters[b],
						selcalLetters[c], selcalLetters[d]))
				}
			}
		}
	}
	return codes
}

// generateUniqueSelcals returns n unique valid SELCAL codes.
func generateUniqueSelcals(n int, rng *rand.Rand) ([]string, error) {
	pool := allValidSelcals()
	if n > len(pool) {
		return nil, fmt.Errorf("requested %d SELCALs but only %d valid codes exist", n, len(pool))
	}
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool[:n], nil
}

// ---- Route-string helpers ----

// combineRouteStrings merges ordered route-segment route strings into a single
// string. When two consecutive segments share a boundary fix (the last token of
// segment N equals the first token of segment N+1) the duplicate is omitted.
func combineRouteStrings(segments []models.RouteSegment) string {
	var parts []string
	for i, seg := range segments {
		tokens := strings.Fields(seg.RouteString)
		if i > 0 && len(parts) > 0 && len(tokens) > 0 {
			if strings.EqualFold(parts[len(parts)-1], tokens[0]) {
				tokens = tokens[1:]
			}
		}
		parts = append(parts, tokens...)
	}
	return strings.Join(parts, " ")
}

// oceanicTrackIdentifier returns the identifier of the track segment (order=1).
// If no track segment exists it returns an empty string.
func oceanicTrackIdentifier(segments []models.RouteSegment, orders map[uint]uint) string {
	for _, seg := range segments {
		if orders[seg.ID] == 1 {
			return seg.Identifier
		}
	}
	return ""
}

// ---- CSV output ----

// buildBookingCSV produces a CSV string from the processed booking rows.
func buildBookingCSV(rows []bookingRow) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{
		"ID", "VATSIM ID", "Departure field", "Arrival field",
		"Oceanic track", "Route", "Take-off time", "Flight level",
		"Domestic flight", "SELCAL code",
	})

	for _, r := range rows {
		_ = w.Write([]string{
			r.id, r.vatsimID, r.departure, r.arrival,
			r.oceanicTrack, r.route, r.takeOffTime, r.flightLevel,
			r.domesticFlight, r.selcalCode,
		})
	}
	w.Flush()
	return buf.String()
}

// ---- CSV parsing ----

func parseBookingCSV(r io.Reader) ([]bookingRow, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}
	if len(header) < 10 {
		return nil, fmt.Errorf("CSV must have at least 10 columns, got %d", len(header))
	}
	// Strip BOM from first header field if present.
	header[0] = strings.TrimPrefix(header[0], "\ufeff")

	var rows []bookingRow
	idx := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row %d: %w", idx+2, err)
		}
		for len(record) < 10 {
			record = append(record, "")
		}
		rows = append(rows, bookingRow{
			idx:            idx,
			id:             strings.TrimSpace(record[0]),
			vatsimID:       strings.TrimSpace(record[1]),
			departure:      strings.TrimSpace(record[2]),
			arrival:        strings.TrimSpace(record[3]),
			oceanicTrack:   strings.TrimSpace(record[4]),
			route:          strings.TrimSpace(record[5]),
			takeOffTime:    strings.TrimSpace(record[6]),
			flightLevel:    strings.TrimSpace(record[7]),
			domesticFlight: strings.TrimSpace(record[8]),
			selcalCode:     strings.TrimSpace(record[9]),
		})
		idx++
	}
	return rows, nil
}

// ---- Main handler ----

// ImportBookingCSV godoc
//
//	@Summary	Import booking CSV and populate oceanic track & route
//	@Description	Accepts a booking-portal CSV file, matches each row to a slot
//	@Description	in the latest slot revision by city pair + departure time, and
//	@Description	fills in the Oceanic Track and Route columns. A mapping of
//	@Description	booking ID → slot ID is persisted for reproducibility.
//	@Description	Use format=file to get the result as a downloadable CSV file,
//	@Description	or omit/set format=json for a JSON response with match stats.
//	@Tags		bookings
//	@Security	ApiKeyAuth
//	@Accept		multipart/form-data
//	@Produce	json,text/csv
//	@Param		eventId		path		int		true	"Event ID"
//	@Param		file		formData	file	true	"Booking CSV file"
//	@Param		domestic	query		bool	false	"Set Domestic flight to false for all rows"
//	@Param		selcal		query		bool	false	"Generate unique valid SELCAL codes"
//	@Param		format		query		string	false	"Response format: 'json' (default) or 'file'"
//	@Success	200		{object}	map[string]interface{}	"JSON with csv string, match stats, unmatched IDs"
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/bookings/import [post]
func ImportBookingCSV(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	domesticFlag := strings.EqualFold(c.Query("domestic"), "true")
	selcalFlag := strings.EqualFold(c.Query("selcal"), "true")

	// --- Read uploaded file ---
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing 'file' form field")
	}
	f, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cannot open uploaded file")
	}
	defer f.Close()

	rows, err := parseBookingCSV(f)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if len(rows) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "CSV contains no data rows")
	}

	// --- Load latest slot revision with route segments ---
	var revision models.SlotRevision
	result := database.DB.
		Preload("Slots.DepartureAirport.Waypoint").
		Preload("Slots.ArrivalAirport.Waypoint").
		Preload("Slots.RouteSegments").
		Where("event_id = ?", eventID).
		Order("number DESC").
		First(&revision)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "no slot revisions found for this event")
	}

	// Sort route segments within each slot by their order.
	sortSlotRouteSegments(revision.Slots)

	// Build order map for all slots so we can identify the track segment (order=1).
	slotIDs := make([]uint, len(revision.Slots))
	for i, s := range revision.Slots {
		slotIDs[i] = s.ID
	}
	var allSRS []models.SlotRouteSegment
	database.DB.Where(`"slot_id" IN ?`, slotIDs).Find(&allSRS)
	// orderBySlot[slotID][routeSegmentID] = order
	orderBySlot := make(map[uint]map[uint]uint)
	for _, srs := range allSRS {
		if orderBySlot[srs.SlotID] == nil {
			orderBySlot[srs.SlotID] = make(map[uint]uint)
		}
		orderBySlot[srs.SlotID][srs.RouteSegmentID] = srs.Order
	}

	// --- Group DB slots by city pair, sorted by departure time ---
	dbSlotsByPair := make(map[cityPair][]*bookingSlotInfo)
	for _, s := range revision.Slots {
		dep := s.DepartureAirport.Waypoint.Identifier
		arr := s.ArrivalAirport.Waypoint.Identifier
		cp := cityPair{dep: dep, arr: arr}
		dbSlotsByPair[cp] = append(dbSlotsByPair[cp], &bookingSlotInfo{
			slot:     s,
			orderMap: orderBySlot[s.ID],
		})
	}
	for cp := range dbSlotsByPair {
		sort.Slice(dbSlotsByPair[cp], func(i, j int) bool {
			return dbSlotsByPair[cp][i].slot.DepartureTime.Before(dbSlotsByPair[cp][j].slot.DepartureTime)
		})
	}

	// --- Load existing mappings to avoid re-matching ---
	var existingMappings []models.BookingSlotMapping
	database.DB.Where("event_id = ? AND slot_revision_id = ?", eventID, revision.ID).Find(&existingMappings)
	existingByBookingID := make(map[uint]models.BookingSlotMapping)
	for _, m := range existingMappings {
		existingByBookingID[m.BookingID] = m
	}

	// --- Group CSV rows by city pair, preserving insertion order ---
	csvByPair := make(map[cityPair][]csvSlotRef)
	for i := range rows {
		cp := cityPair{dep: rows[i].departure, arr: rows[i].arrival}
		csvByPair[cp] = append(csvByPair[cp], csvSlotRef{rowIdx: i, row: &rows[i]})
	}
	// Sort each group by take-off time so ordering matches.
	for cp := range csvByPair {
		sort.Slice(csvByPair[cp], func(i, j int) bool {
			return csvByPair[cp][i].row.takeOffTime < csvByPair[cp][j].row.takeOffTime
		})
	}

	// --- Match CSV rows to DB slots ---
	var matches []bookingMatchResult
	var warnings []string

	for cp, csvRefs := range csvByPair {
		dbSlots := dbSlotsByPair[cp]
		if len(dbSlots) == 0 {
			for _, ref := range csvRefs {
				warnings = append(warnings, fmt.Sprintf(
					"row %s: no DB slots for %s→%s", ref.row.id, cp.dep, cp.arr))
			}
			continue
		}

		// Walk through CSV refs and consume unused DB slots in time order.
		dbIdx := 0
		for _, ref := range csvRefs {
			bookingID, _ := strconv.ParseUint(ref.row.id, 10, 64)
			bid := uint(bookingID)

			if existing, ok := existingByBookingID[bid]; ok {
				var slot models.Slot
				if err := database.DB.Preload("RouteSegments").First(&slot, existing.SlotID).Error; err != nil {
					warnings = append(warnings, fmt.Sprintf(
						"row %s: existing mapping references slot %d which no longer exists", ref.row.id, existing.SlotID))
				} else {
					track := oceanicTrackIdentifier(slot.RouteSegments, orderBySlot[slot.ID])
					route := combineRouteStrings(slot.RouteSegments)
					ref.row.oceanicTrack = track
					ref.row.route = route
					matches = append(matches, bookingMatchResult{
						bookingID: bid,
						slotID:    existing.SlotID,
						track:     track,
						route:     route,
					})
					for i, si := range dbSlots {
						if si.slot.ID == existing.SlotID {
							dbSlots[i].used = true
							break
						}
					}
				}
				continue
			}

			// Find next unused DB slot for this pair.
			for dbIdx < len(dbSlots) && dbSlots[dbIdx].used {
				dbIdx++
			}
			if dbIdx >= len(dbSlots) {
				warnings = append(warnings, fmt.Sprintf(
					"row %s: ran out of DB slots for %s→%s", ref.row.id, cp.dep, cp.arr))
				continue
			}

			si := dbSlots[dbIdx]
			si.used = true
			dbIdx++

			track := oceanicTrackIdentifier(si.slot.RouteSegments, si.orderMap)
			route := combineRouteStrings(si.slot.RouteSegments)

			ref.row.oceanicTrack = track
			ref.row.route = route

			matches = append(matches, bookingMatchResult{
				bookingID: bid,
				slotID:    si.slot.ID,
				track:     track,
				route:     route,
			})
		}
	}

	// --- Optional: domestic flag ---
	if domesticFlag {
		for i := range rows {
			rows[i].domesticFlight = "false"
		}
	}

	// --- Optional: SELCAL generation ---
	if selcalFlag {
		rng := rand.New(rand.NewSource(int64(revision.ID)*1000 + int64(len(rows))))
		codes, err := generateUniqueSelcals(len(rows), rng)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		for i := range rows {
			rows[i].selcalCode = codes[i]
		}
	}

	// --- Persist mapping ---
	if len(matches) > 0 {
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Clear previous mappings for this event+revision.
			tx.Where("event_id = ? AND slot_revision_id = ?", eventID, revision.ID).
				Delete(&models.BookingSlotMapping{})

			mappings := make([]models.BookingSlotMapping, len(matches))
			for i, m := range matches {
				mappings[i] = models.BookingSlotMapping{
					EventID:        uint(eventID),
					SlotRevisionID: revision.ID,
					BookingID:      m.bookingID,
					SlotID:         m.slotID,
				}
			}
			return tx.Create(&mappings).Error
		}); err != nil {
			log.Error().Err(err).Msg("bookings: failed to persist slot mappings")
			return fiber.NewError(fiber.StatusInternalServerError, "failed to save booking-slot mappings")
		}
	}

	// --- Collect unmatched booking IDs ---
	var unmatchedIDs []string
	for _, r := range rows {
		if r.oceanicTrack == "" && r.route == "" {
			unmatchedIDs = append(unmatchedIDs, r.id)
		}
	}

	// --- Build output CSV ---
	csvStr := buildBookingCSV(rows)

	if len(warnings) > 0 {
		log.Warn().Strs("warnings", warnings).Msg("booking import completed with warnings")
	}

	// --- Return file or JSON ---
	if strings.EqualFold(c.Query("format"), "file") {
		c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
		c.Set(fiber.HeaderContentDisposition,
			fmt.Sprintf(`attachment; filename="bookings-event%d.csv"`, eventID))
		return c.SendString(csvStr)
	}

	return c.JSON(fiber.Map{
		"totalRows":    len(rows),
		"matched":      len(matches),
		"unmatched":    len(unmatchedIDs),
		"unmatchedIds": unmatchedIDs,
		"warnings":     warnings,
		"csv":          csvStr,
	})
}

// ImportBookingFromNattrak godoc
//
//	@Summary	Import bookings from Nattrak API and populate oceanic track & route
//	@Description	Fetches booking data from the Nattrak API, matches each row to a slot
//	@Description	in the latest slot revision by city pair + departure time, and
//	@Description	fills in the Oceanic Track and Route columns. A mapping of
//	@Description	booking ID → slot ID is persisted for reproducibility.
//	@Description	Use format=file to get the result as a downloadable CSV file,
//	@Description	or omit/set format=json for a JSON response with match stats.
//	@Tags		bookings
//	@Security	ApiKeyAuth
//	@Produce	json,text/csv
//	@Param		eventId	path	int		true	"Event ID"
//	@Param		domestic	query	bool	false	"Set Domestic flight to false for all rows"
//	@Param		selcal		query	bool	false	"Generate unique valid SELCAL codes"
//	@Param		format		query	string	false	"Response format: 'json' (default) or 'file'"
//	@Success	200		{object}	map[string]interface{}	"JSON with csv string, match stats, unmatched IDs"
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/bookings/import/nattrak [get]
func ImportBookingFromNattrak(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	domesticFlag := strings.EqualFold(c.Query("domestic"), "true")
	selcalFlag := strings.EqualFold(c.Query("selcal"), "true")

	resp, err := http.Get("https://ctp.vatsim.net/api/bookings-nattrak")
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to fetch from Nattrak API: "+err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("Nattrak API returned status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to read Nattrak response: "+err.Error())
	}

	var nattrakResp nattrakResponse
	if err := json.Unmarshal(body, &nattrakResp); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to parse Nattrak response: "+err.Error())
	}

	if len(nattrakResp.Data) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Nattrak API returned no booking data")
	}

	rows := make([]bookingRow, 0, len(nattrakResp.Data))
	for _, nb := range nattrakResp.Data {
		totTime := nb.TOT
		if idx := strings.LastIndex(nb.TOT, " "); idx != -1 {
			totTime = nb.TOT[idx+1:]
		}
		if len(totTime) > 5 {
			totTime = totTime[len(totTime)-5:]
		}

		domestic := "true"
		if !nb.IsDomestic {
			domestic = "false"
		}

		selcal := ""
		if nb.SELCAL != nil {
			selcal = *nb.SELCAL
		}

		rows = append(rows, bookingRow{
			idx:            len(rows),
			id:             strconv.FormatUint(uint64(nb.ID), 10),
			vatsimID:       strconv.FormatUint(uint64(nb.UserID), 10),
			departure:      nb.DepID,
			arrival:        nb.ArrID,
			oceanicTrack:   "",
			route:          "",
			takeOffTime:    totTime,
			flightLevel:    strconv.Itoa(nb.Level),
			domesticFlight: domestic,
			selcalCode:     selcal,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		ti, _ := time.Parse("15:04", rows[i].takeOffTime)
		tj, _ := time.Parse("15:04", rows[j].takeOffTime)
		return ti.Before(tj)
	})

	var revision models.SlotRevision
	result := database.DB.
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

	slotIDs := make([]uint, len(revision.Slots))
	for i, s := range revision.Slots {
		slotIDs[i] = s.ID
	}
	var allSRS []models.SlotRouteSegment
	database.DB.Where(`"slot_id" IN ?`, slotIDs).Find(&allSRS)
	orderBySlot := make(map[uint]map[uint]uint)
	for _, srs := range allSRS {
		if orderBySlot[srs.SlotID] == nil {
			orderBySlot[srs.SlotID] = make(map[uint]uint)
		}
		orderBySlot[srs.SlotID][srs.RouteSegmentID] = srs.Order
	}

	var existingMappings []models.BookingSlotMapping
	database.DB.Where("event_id = ? AND slot_revision_id = ?", eventID, revision.ID).Find(&existingMappings)
	existingByBookingID := make(map[uint]models.BookingSlotMapping)
	for _, m := range existingMappings {
		existingByBookingID[m.BookingID] = m
	}

	dbSlotsByPair := make(map[cityPair][]*bookingSlotInfo)
	for _, s := range revision.Slots {
		dep := s.DepartureAirport.Waypoint.Identifier
		arr := s.ArrivalAirport.Waypoint.Identifier
		cp := cityPair{dep: dep, arr: arr}
		dbSlotsByPair[cp] = append(dbSlotsByPair[cp], &bookingSlotInfo{
			slot:     s,
			orderMap: orderBySlot[s.ID],
		})
	}
	for cp := range dbSlotsByPair {
		sort.Slice(dbSlotsByPair[cp], func(i, j int) bool {
			return dbSlotsByPair[cp][i].slot.DepartureTime.Before(dbSlotsByPair[cp][j].slot.DepartureTime)
		})
	}

	csvByPair := make(map[cityPair][]csvSlotRef)
	for i := range rows {
		cp := cityPair{dep: rows[i].departure, arr: rows[i].arrival}
		csvByPair[cp] = append(csvByPair[cp], csvSlotRef{rowIdx: i, row: &rows[i]})
	}
	for cp := range csvByPair {
		sort.Slice(csvByPair[cp], func(i, j int) bool {
			return csvByPair[cp][i].row.takeOffTime < csvByPair[cp][j].row.takeOffTime
		})
	}

	var matches []bookingMatchResult
	var warnings []string
	matchedBookingIDs := make(map[uint]bool)

	for cp, csvRefs := range csvByPair {
		dbSlots := dbSlotsByPair[cp]
		if len(dbSlots) == 0 {
			for _, ref := range csvRefs {
				warnings = append(warnings, fmt.Sprintf(
					"row %s: no DB slots for %s→%s", ref.row.id, cp.dep, cp.arr))
			}
			continue
		}

		dbIdx := 0
		for _, ref := range csvRefs {
			bookingID, _ := strconv.ParseUint(ref.row.id, 10, 64)
			bid := uint(bookingID)

			if existing, ok := existingByBookingID[bid]; ok {
				var slot models.Slot
				if err := database.DB.Preload("RouteSegments").First(&slot, existing.SlotID).Error; err != nil {
					warnings = append(warnings, fmt.Sprintf(
						"row %s: existing mapping references slot %d which no longer exists", ref.row.id, existing.SlotID))
					continue
				}
				track := oceanicTrackIdentifier(slot.RouteSegments, orderBySlot[slot.ID])
				route := combineRouteStrings(slot.RouteSegments)
				ref.row.oceanicTrack = track
				ref.row.route = route
				matches = append(matches, bookingMatchResult{
					bookingID: bid,
					slotID:    existing.SlotID,
					track:     track,
					route:     route,
				})
				matchedBookingIDs[bid] = true
				for i, si := range dbSlots {
					if si.slot.ID == existing.SlotID {
						dbSlots[i].used = true
						break
					}
				}
				continue
			}

			for dbIdx < len(dbSlots) && dbSlots[dbIdx].used {
				dbIdx++
			}
			if dbIdx >= len(dbSlots) {
				warnings = append(warnings, fmt.Sprintf(
					"row %s: ran out of DB slots for %s→%s", ref.row.id, cp.dep, cp.arr))
				continue
			}

			si := dbSlots[dbIdx]
			si.used = true
			dbIdx++

			track := oceanicTrackIdentifier(si.slot.RouteSegments, si.orderMap)
			route := combineRouteStrings(si.slot.RouteSegments)

			ref.row.oceanicTrack = track
			ref.row.route = route

			matches = append(matches, bookingMatchResult{
				bookingID: bid,
				slotID:    si.slot.ID,
				track:     track,
				route:     route,
			})
			matchedBookingIDs[bid] = true
		}
	}

	if domesticFlag {
		for i := range rows {
			rows[i].domesticFlight = "false"
		}
	}

	if selcalFlag {
		rng := rand.New(rand.NewSource(int64(revision.ID)*1000 + int64(len(rows))))
		codes, err := generateUniqueSelcals(len(rows), rng)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		for i := range rows {
			rows[i].selcalCode = codes[i]
		}
	}

	if len(matches) > 0 {
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			tx.Where("event_id = ? AND slot_revision_id = ?", eventID, revision.ID).
				Delete(&models.BookingSlotMapping{})

			mappings := make([]models.BookingSlotMapping, len(matches))
			for i, m := range matches {
				mappings[i] = models.BookingSlotMapping{
					EventID:        uint(eventID),
					SlotRevisionID: revision.ID,
					BookingID:      m.bookingID,
					SlotID:         m.slotID,
				}
			}
			return tx.Create(&mappings).Error
		}); err != nil {
			log.Error().Err(err).Msg("bookings: failed to persist slot mappings")
			return fiber.NewError(fiber.StatusInternalServerError, "failed to save booking-slot mappings")
		}
	}

	var unmatchedIDs []string
	for _, r := range rows {
		if r.oceanicTrack == "" && r.route == "" {
			unmatchedIDs = append(unmatchedIDs, r.id)
		}
	}

	csvStr := buildBookingCSV(rows)

	if len(warnings) > 0 {
		log.Warn().Strs("warnings", warnings).Msg("nattrak import completed with warnings")
	}

	sort.Slice(rows, func(i, j int) bool {
		idI, _ := strconv.ParseUint(rows[i].id, 10, 64)
		idJ, _ := strconv.ParseUint(rows[j].id, 10, 64)
		return idI < idJ
	})
	csvStr = buildBookingCSV(rows)

	if strings.EqualFold(c.Query("format"), "file") {
		c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
		c.Set(fiber.HeaderContentDisposition,
			fmt.Sprintf(`attachment; filename="bookings-nattrak-event%d.csv"`, eventID))
		return c.SendString(csvStr)
	}

	return c.JSON(fiber.Map{
		"totalRows":    len(rows),
		"matched":      len(matches),
		"unmatched":    len(unmatchedIDs),
		"unmatchedIds": unmatchedIDs,
		"warnings":     warnings,
		"csv":          csvStr,
	})
}
