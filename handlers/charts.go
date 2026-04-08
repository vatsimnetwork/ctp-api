package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

func findLatestRevisionWithSlots(eventID uint64) (*models.SlotRevision, error) {
	var revision models.SlotRevision
	result := database.DB.Raw(
		`SELECT * FROM slot_revisions WHERE event_id = ? AND EXISTS (SELECT 1 FROM slots WHERE slot_revision_id = slot_revisions.id) ORDER BY number DESC LIMIT 1`,
		eventID,
	).Scan(&revision)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &revision, nil
}

// ChartsDepartureAirports godoc
//
//	@Summary	Get departure airport chart data for an event
//	@Tags		charts
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	object	"revisionNumber, airports (identifier, maximumSlots, slotsAllocated, slots[])"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/events/{id}/charts/departure-airports [get]
func ChartsDepartureAirports(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	revision, err := findLatestRevisionWithSlots(id)
	if err != nil {
		return err
	}
	if revision == nil {
		return c.JSON(fiber.Map{"revisionNumber": 0, "airports": []fiber.Map{}})
	}

	// Slot query requires JOIN through airports→waypoints for identifiers; raw SQL is clearest.
	type slotRow struct {
		DepartureAirportID uint      `gorm:"column:departure_airport_id"`
		DepIdent           string    `gorm:"column:dep_ident"`
		ArrIdent           string    `gorm:"column:arr_ident"`
		DepartureTime      time.Time `gorm:"column:departure_time"`
		ProjectedArrival   time.Time `gorm:"column:projected_arrival_time"`
	}
	var slotRows []slotRow
	if err := database.DB.Raw(`
		SELECT s.departure_airport_id,
		       dw.identifier as dep_ident, aw.identifier as arr_ident,
		       s.departure_time, s.projected_arrival_time
		FROM slots s
		JOIN airports da ON da.id = s.departure_airport_id
		JOIN waypoints dw ON dw.id = da.waypoint_id
		JOIN airports aa ON aa.id = s.arrival_airport_id
		JOIN waypoints aw ON aw.id = aa.waypoint_id
		WHERE s.slot_revision_id = ?
		ORDER BY s.departure_time
	`, revision.ID).Scan(&slotRows).Error; err != nil {
		return err
	}

	var airportModels []models.Airport
	if err := database.DB.Select("id, maximum_slots").Where("event_id = ?", id).Find(&airportModels).Error; err != nil {
		return err
	}
	airportMaxSlots := make(map[uint]uint16, len(airportModels))
	for _, a := range airportModels {
		airportMaxSlots[a.ID] = a.MaximumSlots
	}

	type slotEntry struct {
		Dep     string `json:"dep"`
		Arr     string `json:"arr"`
		DepTime string `json:"depTime"`
		ArrTime string `json:"arrTime"`
	}
	type airportEntry struct {
		Identifier     string      `json:"identifier"`
		MaximumSlots   uint16      `json:"maximumSlots"`
		SlotsAllocated int         `json:"slotsAllocated"`
		Slots          []slotEntry `json:"slots"`
	}

	slotsByDep := map[uint][]slotRow{}
	depIdentMap := map[uint]string{}
	for _, row := range slotRows {
		slotsByDep[row.DepartureAirportID] = append(slotsByDep[row.DepartureAirportID], row)
		if _, ok := depIdentMap[row.DepartureAirportID]; !ok {
			depIdentMap[row.DepartureAirportID] = row.DepIdent
		}
	}

	var airports []airportEntry
	for depID, slots := range slotsByDep {
		entries := make([]slotEntry, 0, len(slots))
		for _, s := range slots {
			entries = append(entries, slotEntry{
				Dep:     s.DepIdent,
				Arr:     s.ArrIdent,
				DepTime: s.DepartureTime.UTC().Format("15:04Z"),
				ArrTime: s.ProjectedArrival.UTC().Format("15:04Z"),
			})
		}
		airports = append(airports, airportEntry{
			Identifier:     depIdentMap[depID],
			MaximumSlots:   airportMaxSlots[depID],
			SlotsAllocated: len(slots),
			Slots:          entries,
		})
	}

	for i := 1; i < len(airports); i++ {
		for j := i; j > 0 && airports[j].Identifier < airports[j-1].Identifier; j-- {
			airports[j], airports[j-1] = airports[j-1], airports[j]
		}
	}

	return c.JSON(fiber.Map{
		"revisionNumber": revision.Number,
		"airports":       airports,
	})
}

// ChartsSectors godoc
//
//	@Summary	Get sector throughput chart data for an event
//	@Tags		charts
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	object	"revisionNumber, eventDate, departureTimeWindow, sectors (identifier, maxAcPerHour, hasTimings, totalSlots, buckets[])"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/events/{id}/charts/sectors [get]
func ChartsSectors(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var event models.VATSIMEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	revision, err := findLatestRevisionWithSlots(id)
	if err != nil {
		return err
	}
	if revision == nil {
		return c.JSON(fiber.Map{
			"revisionNumber":      0,
			"eventDate":           event.Date.UTC(),
			"departureTimeWindow": time.Duration(event.DepartureTimeWindow).String(),
			"sectors":             []fiber.Map{},
		})
	}

	var sectors []models.Sector
	if err := database.DB.Where("event_id = ?", id).Find(&sectors).Error; err != nil {
		return err
	}

	var snapshots []models.ThroughputSnapshot
	if err := database.DB.Where("slot_revision_id = ? AND throughput_point_type = ?", revision.ID, "sector").
		Find(&snapshots).Error; err != nil {
		return err
	}

	type snap struct {
		MinuteOffset int
		SlotID       uint
	}
	snapsBySector := map[uint][]snap{}
	for _, s := range snapshots {
		sID := uint(s.ThroughputPointID)
		snapsBySector[sID] = append(snapsBySector[sID], snap{s.MinuteOffset, s.SlotID})
	}

	eventDate := event.Date.UTC()

	type bucketEntry struct {
		Label string `json:"label"`
		Count int    `json:"count"`
	}
	type sectorEntry struct {
		Identifier   string        `json:"identifier"`
		MaxAcPerHour uint16        `json:"maxAcPerHour"`
		HasTimings   bool          `json:"hasTimings"`
		TotalSlots   int           `json:"totalSlots"`
		Buckets      []bucketEntry `json:"buckets"`
	}

	result := make([]sectorEntry, 0, len(sectors))
	for _, s := range sectors {
		snaps := snapsBySector[s.ID]
		hasTimings := len(snaps) > 0

		bucketSlots := map[int]map[uint]bool{}
		allSlotIDs := map[uint]bool{}
		for _, sn := range snaps {
			bucket := floorDivCharts(sn.MinuteOffset, 20)
			if bucketSlots[bucket] == nil {
				bucketSlots[bucket] = map[uint]bool{}
			}
			bucketSlots[bucket][sn.SlotID] = true
			allSlotIDs[sn.SlotID] = true
		}

		var buckets []bucketEntry
		if hasTimings {
			minBucket, maxBucket := int(^uint(0)>>1), -int(^uint(0)>>1)-1
			for b := range bucketSlots {
				if b < minBucket {
					minBucket = b
				}
				if b > maxBucket {
					maxBucket = b
				}
			}
			for b := minBucket; b <= maxBucket; b++ {
				label := eventDate.Add(time.Duration(b*20) * time.Minute).UTC().Format("15:04Z")
				buckets = append(buckets, bucketEntry{Label: label, Count: len(bucketSlots[b])})
			}
		}

		result = append(result, sectorEntry{
			Identifier:   s.Identifier,
			MaxAcPerHour: s.MaximumAircraftPerHour,
			HasTimings:   hasTimings,
			TotalSlots:   len(allSlotIDs),
			Buckets:      buckets,
		})
	}

	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Identifier < result[j-1].Identifier; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	return c.JSON(fiber.Map{
		"revisionNumber":      revision.Number,
		"eventDate":           event.Date.UTC(),
		"departureTimeWindow": time.Duration(event.DepartureTimeWindow).String(),
		"sectors":             result,
	})
}

// ChartsArrivalAirports godoc
//
//	@Summary	Get arrival airport chart data for an event
//	@Tags		charts
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	object	"revisionNumber, eventDate, airports (identifier, maximumSlots, labels[], total[], depSeries[])"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/events/{id}/charts/arrival-airports [get]
func ChartsArrivalAirports(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var event models.VATSIMEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	revision, err := findLatestRevisionWithSlots(id)
	if err != nil {
		return err
	}
	if revision == nil {
		return c.JSON(fiber.Map{
			"revisionNumber": 0,
			"eventDate":      event.Date.UTC(),
			"airports":       []fiber.Map{},
		})
	}

	type arrSlotRow struct {
		ArrivalAirportID     uint      `gorm:"column:arrival_airport_id"`
		ProjectedArrivalTime time.Time `gorm:"column:projected_arrival_time"`
		DepIdent             string    `gorm:"column:dep_ident"`
	}
	var arrSlotRows []arrSlotRow
	if err := database.DB.Raw(`
		SELECT s.arrival_airport_id, s.projected_arrival_time,
		       dw.identifier as dep_ident
		FROM slots s
		JOIN airports da ON da.id = s.departure_airport_id
		JOIN waypoints dw ON dw.id = da.waypoint_id
		WHERE s.slot_revision_id = ?
	`, revision.ID).Scan(&arrSlotRows).Error; err != nil {
		return err
	}

	var airportModels []models.Airport
	if err := database.DB.Preload("Waypoint").Where("event_id = ?", id).Find(&airportModels).Error; err != nil {
		return err
	}
	airportIdent := make(map[uint]string, len(airportModels))
	airportMaxSlots := make(map[uint]uint16, len(airportModels))
	for _, a := range airportModels {
		airportIdent[a.ID] = a.Waypoint.Identifier
		airportMaxSlots[a.ID] = a.MaximumSlots
	}

	eventDate := event.Date.UTC()

	arrDepBuckets := map[uint]map[string]map[int]int{}
	for _, row := range arrSlotRows {
		if row.DepIdent == "" {
			continue
		}
		bucket := floorDivCharts(int(row.ProjectedArrivalTime.Sub(eventDate).Minutes()), 20)
		arrID := row.ArrivalAirportID
		if arrDepBuckets[arrID] == nil {
			arrDepBuckets[arrID] = map[string]map[int]int{}
		}
		if arrDepBuckets[arrID][row.DepIdent] == nil {
			arrDepBuckets[arrID][row.DepIdent] = map[int]int{}
		}
		arrDepBuckets[arrID][row.DepIdent][bucket]++
	}

	arrIDs := make([]uint, 0, len(arrDepBuckets))
	for id := range arrDepBuckets {
		arrIDs = append(arrIDs, id)
	}
	for i := 1; i < len(arrIDs); i++ {
		for j := i; j > 0 && airportIdent[arrIDs[j]] < airportIdent[arrIDs[j-1]]; j-- {
			arrIDs[j], arrIDs[j-1] = arrIDs[j-1], arrIDs[j]
		}
	}

	var depColors = []string{
		"#3b82f6", "#f59e0b", "#10b981", "#ef4444", "#8b5cf6",
		"#06b6d4", "#f97316", "#84cc16", "#ec4899", "#6366f1",
		"#14b8a6", "#f43f5e",
	}

	type depSeriesEntry struct {
		Dep     string `json:"dep"`
		Color   string `json:"color"`
		Buckets []int  `json:"buckets"`
	}
	type airportEntry struct {
		Identifier   string           `json:"identifier"`
		MaximumSlots uint16           `json:"maximumSlots"`
		Labels       []string         `json:"labels"`
		Total        []int            `json:"total"`
		DepSeries    []depSeriesEntry `json:"depSeries"`
	}

	var airports []airportEntry
	for _, arrID := range arrIDs {
		ident := airportIdent[arrID]
		if ident == "" {
			continue
		}
		depMap := arrDepBuckets[arrID]

		minBucket, maxBucket := int(^uint(0)>>1), -int(^uint(0)>>1)-1
		for _, buckets := range depMap {
			for b := range buckets {
				if b < minBucket {
					minBucket = b
				}
				if b > maxBucket {
					maxBucket = b
				}
			}
		}
		if minBucket > maxBucket {
			continue
		}

		length := maxBucket - minBucket + 1
		labels := make([]string, length)
		for i := range labels {
			b := minBucket + i
			labels[i] = eventDate.Add(time.Duration(b*20) * time.Minute).UTC().Format("15:04Z")
		}

		// Sort dep identifiers alphabetically for consistent color assignment.
		depIdents := make([]string, 0, len(depMap))
		for d := range depMap {
			depIdents = append(depIdents, d)
		}
		for i := 1; i < len(depIdents); i++ {
			for j := i; j > 0 && depIdents[j] < depIdents[j-1]; j-- {
				depIdents[j], depIdents[j-1] = depIdents[j-1], depIdents[j]
			}
		}

		total := make([]int, length)
		var depSeries []depSeriesEntry
		for ci, dep := range depIdents {
			buckets := depMap[dep]
			series := make([]int, length)
			for b, count := range buckets {
				idx := b - minBucket
				series[idx] = count
				total[idx] += count
			}
			depSeries = append(depSeries, depSeriesEntry{
				Dep:     dep,
				Color:   depColors[ci%len(depColors)],
				Buckets: series,
			})
		}

		airports = append(airports, airportEntry{
			Identifier:   ident,
			MaximumSlots: airportMaxSlots[arrID],
			Labels:       labels,
			Total:        total,
			DepSeries:    depSeries,
		})
	}

	return c.JSON(fiber.Map{
		"revisionNumber": revision.Number,
		"eventDate":      event.Date.UTC(),
		"airports":       airports,
	})
}

func floorDivCharts(a, b int) int {
	q := a / b
	if (a^b) < 0 && q*b != a {
		q--
	}
	return q
}
