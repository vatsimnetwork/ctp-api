package handlers

import (
	"fmt"
	"math"
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
	if err := database.DB.Where("event_id = ? OR event_id IS NULL", id).Find(&sectors).Error; err != nil {
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

	syncBase := event.Date.UTC()
	if tod := event.DepartureTimeWindowOffsetSynchronizationTimeOfDay; tod != "" {
		var h, m int
		fmt.Sscanf(tod, "%d:%d", &h, &m)
		syncBase = syncBase.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	}

	type bucketEntry struct {
		Label       string `json:"label"`
		PeakCount   int    `json:"peakCount"`
		UniqueCount int    `json:"uniqueCount"`
	}
	type sectorEntry struct {
		Identifier              string        `json:"identifier"`
		MaxAcPerHour            uint16        `json:"maxAcPerHour"`
		HasTimings              bool          `json:"hasTimings"`
		TotalSlots              int           `json:"totalSlots"`
		AvgDwellMinutes         float64       `json:"avgDwellMinutes"`
		EstimatedMaxOccupancy   int           `json:"estimatedMaxOccupancy"`
		EstimatedTotalOccupancy int           `json:"estimatedTotalOccupancy"`
		Buckets                 []bucketEntry `json:"buckets"`
	}

	// The simulator step size is stored directly on the event — no need to infer it from data gaps.
	analysisResolution := int(event.SimulationAnalysisResolutionInMinutes)
	if analysisResolution == 0 {
		analysisResolution = 2 // safe fallback if the field was never set
	}

	result := make([]sectorEntry, 0, len(sectors))
	for _, s := range sectors {
		snaps := snapsBySector[s.ID]
		hasTimings := len(snaps) > 0

		// --- Step 2: Build 20-minute occupancy buckets ---
		//
		// Each snapshot row says "slot X was inside this sector at minute Y" (a single tick).
		// We first count unique slots at each individual tick (the true instantaneous occupancy),
		// then for each 20-minute window we take the PEAK tick count within that window.
		//
		// Why peak-per-tick rather than unique slots across the whole window?
		// If a sector has a short dwell time (say 8 min), many aircraft enter and leave within
		// a single 20-min window, so the cumulative unique count would be far higher than the
		// number ever present at one moment — making it incomparable to Little's Law's
		// instantaneous occupancy estimate. Peak-per-tick stays in the same unit.
		tickSlots := map[int]map[uint]bool{}
		for _, sn := range snaps {
			if tickSlots[sn.MinuteOffset] == nil {
				tickSlots[sn.MinuteOffset] = map[uint]bool{}
			}
			tickSlots[sn.MinuteOffset][sn.SlotID] = true
		}

		// For each 20-minute bucket, find the highest single-tick count (peak instantaneous
		// occupancy) and also the cumulative unique slots across the window (total flow).
		bucketPeak := map[int]int{}
		bucketUnique := map[int]map[uint]bool{}
		for offset, slots := range tickSlots {
			bucket := floorDivCharts(offset, 20)
			if count := len(slots); count > bucketPeak[bucket] {
				bucketPeak[bucket] = count
			}
			if bucketUnique[bucket] == nil {
				bucketUnique[bucket] = map[uint]bool{}
			}
			for id := range slots {
				bucketUnique[bucket][id] = true
			}
		}

		var buckets []bucketEntry
		if hasTimings {
			minBucket, maxBucket := int(^uint(0)>>1), -int(^uint(0)>>1)-1
			for b := range bucketPeak {
				if b < minBucket {
					minBucket = b
				}
				if b > maxBucket {
					maxBucket = b
				}
			}
			for b := minBucket; b <= maxBucket; b++ {
				label := syncBase.Add(time.Duration(b*20) * time.Minute).UTC().Format("15:04Z")
				buckets = append(buckets, bucketEntry{
					Label:       label,
					PeakCount:   bucketPeak[b],
					UniqueCount: len(bucketUnique[b]),
				})
			}
		}

		// --- Step 3: Estimate maximum sector occupancy via Little's Law ---
		//
		// The problem: maximumAircraftPerHour is a SLOT ASSIGNMENT RATE (e.g. "assign at most
		// 30 slots/hr to routes through this sector"). The chart shows OCCUPANCY — how many
		// aircraft are physically present in the sector at the same time. These are different
		// units and cannot be directly compared.
		//
		// Little's Law bridges the gap:
		//   Average Occupancy = Arrival Rate × Average Dwell Time
		//   (aircraft present) = (aircraft/hr) × (hours each one stays)
		//
		// We derive average dwell time per-slot from the simulation snapshots:
		//   For each slot, find the first and last minute it was seen in this sector.
		//   Its dwell time = (lastMinute - firstMinute + analysisResolution).
		//   The +resolution accounts for the fact that the aircraft was still present
		//   during the last recorded tick, so it stayed at least one more step.
		//
		//   Example: slot seen at minutes 10, 12, 14 with resolution=2
		//     → dwell = 14 - 10 + 2 = 6 minutes
		//
		//   We then average those individual dwell times across all slots.
		//
		//   estimatedMaxOccupancy = maxAcPerHour × avgDwellMinutes / 60
		//
		//   Example: avgDwellMinutes=12, maxAcPerHour=30
		//     → 30 × (12/60) = 6 aircraft concurrently at capacity
		//
		// For unlimited sectors (65535) we skip — no meaningful capacity limit to draw.
		// For sectors with no simulation data we also skip — nothing to derive dwell time from.
		type slotRange struct{ min, max int }
		perSlot := map[uint]slotRange{}
		for _, sn := range snaps {
			if r, ok := perSlot[sn.SlotID]; ok {
				if sn.MinuteOffset < r.min {
					r.min = sn.MinuteOffset
				}
				if sn.MinuteOffset > r.max {
					r.max = sn.MinuteOffset
				}
				perSlot[sn.SlotID] = r
			} else {
				perSlot[sn.SlotID] = slotRange{sn.MinuteOffset, sn.MinuteOffset}
			}
		}

		avgDwellMinutes := 0.0
		estimatedMaxOccupancy := 0
		estimatedTotalOccupancy := 0
		if hasTimings && len(perSlot) > 0 {
			totalDwell := 0
			for _, r := range perSlot {
				totalDwell += r.max - r.min + analysisResolution
			}
			avgDwellMinutes = math.Round(float64(totalDwell)/float64(len(perSlot))*10) / 10
			if s.MaximumAircraftPerHour > 0 && s.MaximumAircraftPerHour < 65535 {
				// Max occupancy (instantaneous): λ × W / 60
				estimatedMaxOccupancy = int(math.Round(float64(s.MaximumAircraftPerHour) * avgDwellMinutes / 60.0))
				// Total occupancy (unique aircraft in any 20-min window): λ × (W + 20) / 60
				// = steady-state occupancy at window start + new arrivals during the window.
				estimatedTotalOccupancy = int(math.Round(float64(s.MaximumAircraftPerHour) * (avgDwellMinutes + 20) / 60.0))
			}
		}

		result = append(result, sectorEntry{
			Identifier:              s.Identifier,
			MaxAcPerHour:            s.MaximumAircraftPerHour,
			HasTimings:              hasTimings,
			TotalSlots:              len(perSlot),
			AvgDwellMinutes:         avgDwellMinutes,
			EstimatedMaxOccupancy:   estimatedMaxOccupancy,
			EstimatedTotalOccupancy: estimatedTotalOccupancy,
			Buckets:                 buckets,
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

// ChartsSectorFine godoc
//
//	@Summary	Get fine-grained (2-minute) sector data for a single sector
//	@Tags		charts
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id			path	int		true	"Event ID"
//	@Param		identifier	path	string	true	"Sector Identifier"
//	@Success	200			{object}	object	"labels[], data[] (2-minute intervals)"
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Failure	500			{object}	models.ErrorResponse
//	@Router		/events/{id}/charts/sector/{identifier}/fine [get]
func ChartsSectorFine(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}
	identifier := c.Params("identifier")

	var event models.VATSIMEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	revision, err := findLatestRevisionWithSlots(id)
	if err != nil {
		return err
	}
	if revision == nil {
		return c.JSON(fiber.Map{"labels": []string{}, "data": []int{}})
	}

	var sector models.Sector
	if err := database.DB.Where("identifier = ? AND (event_id = ? OR event_id IS NULL)", identifier, id).First(&sector).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "sector not found")
	}

	var snapshots []models.ThroughputSnapshot
	if err := database.DB.Where("slot_revision_id = ? AND throughput_point_type = ? AND throughput_point_id = ?",
		revision.ID, "sector", sector.ID).
		Find(&snapshots).Error; err != nil {
		return err
	}

	syncBase := event.Date.UTC()
	if tod := event.DepartureTimeWindowOffsetSynchronizationTimeOfDay; tod != "" {
		var h, m int
		fmt.Sscanf(tod, "%d:%d", &h, &m)
		syncBase = syncBase.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	}

	tickSlots := map[int]map[uint]bool{}
	for _, sn := range snapshots {
		if tickSlots[sn.MinuteOffset] == nil {
			tickSlots[sn.MinuteOffset] = map[uint]bool{}
		}
		tickSlots[sn.MinuteOffset][sn.SlotID] = true
	}

	if len(tickSlots) == 0 {
		return c.JSON(fiber.Map{"labels": []string{}, "data": []int{}})
	}

	minOffset, maxOffset := 0, 0
	for offset := range tickSlots {
		if offset < minOffset {
			minOffset = offset
		}
		if offset > maxOffset {
			maxOffset = offset
		}
	}

	analysisResolution := int(event.SimulationAnalysisResolutionInMinutes)
	if analysisResolution == 0 {
		analysisResolution = 2
	}

	labels := make([]string, 0, (maxOffset-minOffset)/analysisResolution+1)
	data := make([]int, 0, (maxOffset-minOffset)/analysisResolution+1)
	for offset := minOffset; offset <= maxOffset; offset += analysisResolution {
		count := 0
		if slots, ok := tickSlots[offset]; ok {
			count = len(slots)
		}
		labels = append(labels, syncBase.Add(time.Duration(offset)*time.Minute).UTC().Format("15:04Z"))
		data = append(data, count)
	}

	return c.JSON(fiber.Map{"labels": labels, "data": data})
}

// ChartsArrivalFine godoc
//
//	@Summary	Get fine-grained (2-minute) arrival data for a single airport
//	@Tags		charts
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id			path	int		true	"Event ID"
//	@Param		identifier	path	string	true	"Arrival Airport Identifier"
//	@Success	200			{object}	object	"labels[], total[], depSeries[]"
//	@Failure	400			{object}	models.ErrorResponse
//	@Failure	404			{object}	models.ErrorResponse
//	@Failure	500			{object}	models.ErrorResponse
//	@Router		/events/{id}/charts/arrival/{identifier}/fine [get]
func ChartsArrivalFine(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}
	identifier := c.Params("identifier")

	var event models.VATSIMEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	revision, err := findLatestRevisionWithSlots(id)
	if err != nil {
		return err
	}
	if revision == nil {
		return c.JSON(fiber.Map{"labels": []string{}, "total": []int{}, "depSeries": []fiber.Map{}})
	}

	var airport models.Airport
	if err := database.DB.Preload("Waypoint").
		Joins("JOIN waypoints ON waypoints.id = airports.waypoint_id AND waypoints.identifier = ?", identifier).
		Where("airports.event_id = ?", id).
		First(&airport).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "arrival airport not found")
	}

	type arrSlotRow struct {
		ProjectedArrivalTime time.Time `gorm:"column:projected_arrival_time"`
		DepIdent             string    `gorm:"column:dep_ident"`
	}
	var arrSlotRows []arrSlotRow
	if err := database.DB.Raw(`
		SELECT s.projected_arrival_time, dw.identifier as dep_ident
		FROM slots s
		JOIN airports da ON da.id = s.departure_airport_id
		JOIN waypoints dw ON dw.id = da.waypoint_id
		WHERE s.slot_revision_id = ? AND s.arrival_airport_id = ?
	`, revision.ID, airport.ID).Scan(&arrSlotRows).Error; err != nil {
		return err
	}

	eventDate := event.Date.UTC()
	analysisResolution := int(event.SimulationAnalysisResolutionInMinutes)
	if analysisResolution == 0 {
		analysisResolution = 2
	}

	type tickEntry struct {
		offset   int
		depIdent string
	}
	var allTicks []tickEntry
	for _, row := range arrSlotRows {
		offset := int(row.ProjectedArrivalTime.Sub(eventDate).Minutes())
		if row.DepIdent == "" {
			continue
		}
		allTicks = append(allTicks, tickEntry{offset: offset, depIdent: row.DepIdent})
	}

	if len(allTicks) == 0 {
		return c.JSON(fiber.Map{"labels": []string{}, "total": []int{}, "depSeries": []fiber.Map{}})
	}

	minOffset, maxOffset := allTicks[0].offset, allTicks[0].offset
	for _, t := range allTicks {
		if t.offset < minOffset {
			minOffset = t.offset
		}
		if t.offset > maxOffset {
			maxOffset = t.offset
		}
	}

	depOffsets := map[string]map[int]int{}
	for _, t := range allTicks {
		if depOffsets[t.depIdent] == nil {
			depOffsets[t.depIdent] = map[int]int{}
		}
		roundedOffset := (t.offset / analysisResolution) * analysisResolution
		depOffsets[t.depIdent][roundedOffset]++
	}

	depColors := []string{
		"#3b82f6", "#f59e0b", "#10b981", "#ef4444", "#8b5cf6",
		"#06b6d4", "#f97316", "#84cc16", "#ec4899", "#6366f1",
		"#14b8a6", "#f43f5e",
	}

	depIdents := make([]string, 0, len(depOffsets))
	for d := range depOffsets {
		depIdents = append(depIdents, d)
	}
	for i := 1; i < len(depIdents); i++ {
		for j := i; j > 0 && depIdents[j] < depIdents[j-1]; j-- {
			depIdents[j], depIdents[j-1] = depIdents[j-1], depIdents[j]
		}
	}

	numPoints := (maxOffset-minOffset)/analysisResolution + 1
	labels := make([]string, numPoints)
	for i := 0; i < numPoints; i++ {
		offset := minOffset + i*analysisResolution
		labels[i] = eventDate.Add(time.Duration(offset) * time.Minute).UTC().Format("15:04Z")
	}

	total := make([]int, numPoints)
	var depSeries []fiber.Map
	for ci, dep := range depIdents {
		series := make([]int, numPoints)
		for offset, count := range depOffsets[dep] {
			idx := (offset - minOffset) / analysisResolution
			if idx >= 0 && idx < numPoints {
				series[idx] = count
				total[idx] += count
			}
		}
		depSeries = append(depSeries, fiber.Map{
			"dep":     dep,
			"color":   depColors[ci%len(depColors)],
			"buckets": series,
		})
	}

	return c.JSON(fiber.Map{
		"labels":    labels,
		"total":     total,
		"depSeries": depSeries,
	})
}

func floorDivCharts(a, b int) int {
	q := a / b
	if (a^b) < 0 && q*b != a {
		q--
	}
	return q
}
