package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/vatsimnetwork/ctp-api/config"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

const simulatorTimeout = 2 * time.Minute

// simStatusStore maps eventID (uint64) → current status string.
// Updated at key points during simulation so the frontend can poll.
var simStatusStore sync.Map

// simLatestResponseStore maps eventID (uint64) → raw JSON bytes of the last simulator response.
var simLatestResponseStore sync.Map

// GetSimulateStatus godoc
//
//	@Summary	Get current simulation status for an event
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	object	"{ status: string }"
//	@Failure	400	{object}	models.ErrorResponse
//	@Router		/events/{id}/simulate-status [get]
func GetSimulateStatus(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}
	if status, ok := simStatusStore.Load(id); ok {
		return c.JSON(fiber.Map{"status": status})
	}
	return c.JSON(fiber.Map{"status": "idle"})
}

// GetLatestSimulatorResponse godoc
//
//	@Summary	Get the raw JSON body of the most recent simulator response for an event
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	any
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id}/latest-simulator-response [get]
func GetLatestSimulatorResponse(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}
	raw, ok := simLatestResponseStore.Load(id)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "no simulator response available for this event")
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Send(raw.([]byte))
}

func mapAirport(a models.Airport) simAirport {
	var dtws *string
	if a.DepartureTimeWindowStart != nil {
		s := a.DepartureTimeWindowStart.UTC().Format(time.RFC3339)
		dtws = &s
	}
	return simAirport{
		Id:                       a.WaypointID,
		Identifier:               a.Waypoint.Identifier,
		MaximumAircraftPerHour:   a.MaximumAircraftPerHour,
		MaximumSlots:             a.MaximumSlots,
		Latitude:                 a.Waypoint.Latitude,
		Longitude:                a.Waypoint.Longitude,
		NumberOfVotes:            a.NumberOfVotes,
		DepartureTimeWindowStart: dtws,
	}
}

func mapRouteSegment(r models.RouteSegment, airportWaypointIDs map[int64]bool, departureHours float64) simRouteSegment {
	tagIDs := make([]uint, 0, len(r.Tags))
	for _, t := range r.Tags {
		if t.TagID != nil {
			tagIDs = append(tagIDs, *t.TagID)
		}
	}

	sorted := make([]models.Location, len(r.Locations))
	copy(sorted, r.Locations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SortOrder < sorted[j].SortOrder })

	locs := make([]int64, 0, len(sorted))
	for _, l := range sorted {
		locs = append(locs, l.WaypointID)
	}

	pfp := make([]uint, 0, len(r.ProvidedFacilityProgression))
	for _, s := range r.ProvidedFacilityProgression {
		pfp = append(pfp, s.ID)
	}

	var rsMaxSlots uint16
	if r.MaximumAircraftPerHour >= 65535 {
		rsMaxSlots = 65535
	} else {
		computed := uint32(float64(r.MaximumAircraftPerHour) * departureHours)
		if computed > 65535 {
			computed = 65535
		}
		rsMaxSlots = uint16(computed)
	}

	return simRouteSegment{
		Id:                          r.ID,
		Identifier:                  r.Identifier,
		MaximumAircraftPerHour:      r.MaximumAircraftPerHour,
		MaximumSlots:                rsMaxSlots,
		RouteString:                 r.RouteString,
		RouteSegmentGroup:           r.RouteSegmentGroup,
		Color:                       r.Color,
		Enabled:                     r.Enabled,
		RouteSegmentTagIds:          tagIDs,
		ProvidedFacilityProgression: pfp,
		Locations:                   locs,
		RouteRevision:               r.RouteRevision,
	}
}

func mapSlot(s models.Slot) simSlot {
	rsIDs := make([]uint, 0, len(s.RouteSegments))
	for _, rs := range s.RouteSegments {
		rsIDs = append(rsIDs, rs.ID)
	}
	depTime := s.DepartureTime
	arrTime := s.ProjectedArrivalTime
	if depTime.IsZero() {
		depTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if arrTime.IsZero() {
		arrTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return simSlot{
		Id:                   s.ID,
		DepartureTime:        depTime.UTC().Format(time.RFC3339),
		ProjectedArrivalTime: arrTime.UTC().Format(time.RFC3339),
		DepartureAirport:     s.DepartureAirport.WaypointID,
		ArrivalAirport:       s.ArrivalAirport.WaypointID,
		RouteSegments:        rsIDs,
	}
}

func normalizeTimeOfDay(s string) string {
	if strings.Count(s, ":") == 1 {
		return s + ":00"
	}
	return s
}

func formatDepartureTimeWindow(d models.Duration) string {
	total := time.Duration(d)
	h := int(total.Hours())
	m := int(total.Minutes()) % 60
	s := int(total.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func buildSimEvent(event models.VATSIMEvent, revision *models.SlotRevision, includeSlots bool) simEvent {
	departureHours := time.Duration(event.DepartureTimeWindow).Hours()

	airports := make([]simAirport, 0, len(event.Airports))
	airportWaypointIDs := make(map[int64]bool, len(event.Airports))
	// Map DB airport ID -> waypoint ID for deferred pair conversion
	dbToWaypoint := make(map[int64]int64, len(event.Airports))
	for _, a := range event.Airports {
		airports = append(airports, mapAirport(a))
		airportWaypointIDs[a.WaypointID] = true
		dbToWaypoint[int64(a.ID)] = a.WaypointID
	}

	// Load departure pair preferences from the database
	var dbPrefs []models.DeparturePairPreference
	database.DB.Where("event_id = ?", event.ID).Find(&dbPrefs)
	deferredPairs := make([][]int64, 0)
	preferredPairs := make([][]int64, 0)
	for _, p := range dbPrefs {
		waypointPair := []int64{int64(p.DepartureAirportID), int64(p.ArrivalAirportID)}
		if p.Preference == models.PreferenceDeferred {
			deferredPairs = append(deferredPairs, waypointPair)
		} else if p.Preference == models.PreferencePreferred {
			preferredPairs = append(preferredPairs, waypointPair)
		}
	}

	waypointByID := make(map[int64]simWaypoint)
	for _, r := range event.RouteSegments {
		for _, l := range r.Locations {
			if airportWaypointIDs[l.WaypointID] {
				continue
			}
			if _, exists := waypointByID[l.WaypointID]; !exists {
				var wpMaxSlots uint16
				if l.Waypoint.MaximumAircraftPerHour >= 65535 {
					wpMaxSlots = 65535
				} else {
					computed := uint32(float64(l.Waypoint.MaximumAircraftPerHour) * departureHours)
					if computed > 65535 {
						computed = 65535
					}
					wpMaxSlots = uint16(computed)
				}
				waypointByID[l.WaypointID] = simWaypoint{
					Id:                     l.WaypointID,
					Identifier:             l.Waypoint.Identifier,
					MaximumAircraftPerHour: l.Waypoint.MaximumAircraftPerHour,
					MaximumSlots:           wpMaxSlots,
					Latitude:               l.Waypoint.Latitude,
					Longitude:              l.Waypoint.Longitude,
				}
			}
		}
	}
	waypoints := make([]simWaypoint, 0, len(waypointByID))
	for _, w := range waypointByID {
		waypoints = append(waypoints, w)
	}
	sort.Slice(waypoints, func(i, j int) bool { return waypoints[i].Id < waypoints[j].Id })

	routeSegments := make([]simRouteSegment, 0, len(event.RouteSegments))
	for _, r := range event.RouteSegments {
		routeSegments = append(routeSegments, mapRouteSegment(r, airportWaypointIDs, departureHours))
	}

	sectorByID := make(map[uint]simSector)
	for _, r := range event.RouteSegments {
		for _, s := range r.ProvidedFacilityProgression {
			if _, exists := sectorByID[s.ID]; !exists {
				// 65535 means unlimited (sentinel); pin at 65535 if computed exceeds it.
				var maxSlots uint16
				if s.MaximumAircraftPerHour >= 65535 {
					maxSlots = 65535
				} else {
					computed := uint32(float64(s.MaximumAircraftPerHour) * departureHours)
					if computed > 65535 {
						computed = 65535
					}
					maxSlots = uint16(computed)
				}
				sectorByID[s.ID] = simSector{
					Id:                     s.ID,
					Identifier:             s.Identifier,
					MaximumAircraftPerHour: s.MaximumAircraftPerHour,
					MaximumSlots:           maxSlots,
				}
			}
		}
	}
	sectors := make([]simSector, 0, len(sectorByID))
	for _, s := range sectorByID {
		sectors = append(sectors, s)
	}
	sort.Slice(sectors, func(i, j int) bool { return sectors[i].Id < sectors[j].Id })

	// Build tagLimits from all EventTag records on the event's route segments.
	// Key by ID so each EventTag appears exactly once regardless of how many route segments share it.
	type tagEntry struct {
		id         uint
		name       string
		maxPerHour *uint16
	}
	tagLimitByID := make(map[uint]tagEntry)
	for _, r := range event.RouteSegments {
		for _, t := range r.Tags {
			if t.TagID == nil {
				continue
			}
			if _, exists := tagLimitByID[*t.TagID]; !exists {
				tagLimitByID[*t.TagID] = tagEntry{
					id:         *t.TagID,
					name:       t.TagRef.Name,
					maxPerHour: t.TagRef.MaximumAircraftPerHour,
				}
			}
		}
	}
	tagLimits := make([]simTagLimit, 0, len(tagLimitByID))
	for _, te := range tagLimitByID {
		var maxSlots uint16
		if te.maxPerHour == nil || *te.maxPerHour >= 65535 {
			maxSlots = 65535
		} else {
			computed := uint32(float64(*te.maxPerHour) * departureHours)
			if computed > 65535 {
				computed = 65535
			}
			maxSlots = uint16(computed)
		}
		tagLimits = append(tagLimits, simTagLimit{
			Id:           te.id,
			Tag:          te.name,
			MaximumSlots: maxSlots,
		})
	}
	sort.Slice(tagLimits, func(i, j int) bool { return tagLimits[i].Id < tagLimits[j].Id })

	slots := []simSlot{}
	var slotRevisionNumber uint
	if revision != nil {
		slotRevisionNumber = revision.Number
		if includeSlots {
			for _, s := range revision.Slots {
				slots = append(slots, mapSlot(s))
			}
		}
	}

	// Convert deferred pairs from DB airport IDs to waypoint IDs
	convertedDeferredPairs := make([][]int64, 0, len(deferredPairs))
	for _, pair := range deferredPairs {
		if len(pair) == 2 {
			dw, dok := dbToWaypoint[pair[0]]
			aw, aok := dbToWaypoint[pair[1]]
			if dok && aok {
				convertedDeferredPairs = append(convertedDeferredPairs, []int64{dw, aw})
			}
		}
	}

	// Convert preferred pairs from DB airport IDs to waypoint IDs
	convertedPreferredPairs := make([][]int64, 0, len(preferredPairs))
	for _, pair := range preferredPairs {
		if len(pair) == 2 {
			dw, dok := dbToWaypoint[pair[0]]
			aw, aok := dbToWaypoint[pair[1]]
			if dok && aok {
				convertedPreferredPairs = append(convertedPreferredPairs, []int64{dw, aw})
			}
		}
	}

	return simEvent{
		Id:                  event.ID,
		Title:               event.Title,
		RouteRevision:       event.RouteRevision,
		SlotRevision:        slotRevisionNumber,
		Date:                event.Date.Format("2006-01-02"),
		DepartureTimeWindow: formatDepartureTimeWindow(event.DepartureTimeWindow),
		CalculationParameters: simCalculationParameters{
			IntendedSlotGenerationMode:                            uint(event.IntendedSlotGenerationMode),
			DepartureTimeWindowOffsetSynchronizationLongitude:     event.DepartureTimeWindowOffsetSynchronizationLongitude,
			SimulationAnalysisResolutionInMinutes:                 event.SimulationAnalysisResolutionInMinutes,
			ShouldSimulationUseActualWeatherForecastData:          event.ShouldSimulationUseActualWeatherForecastData,
			IntendedDepartureTimeWindowOffsetsCalculationMode:     uint(event.IntendedDepartureTimeWindowOffsetsCalculationMode),
			DepartureTimeWindowOffsetSynchronizationTimeOfDay:     normalizeTimeOfDay(event.DepartureTimeWindowOffsetSynchronizationTimeOfDay),
			CalculateThroughputDataOnlyForManuallyProvidedSectors: event.CalculateThroughputDataOnlyForManuallyProvidedSectors,
			IntendedWaypointThroughputCalculationMode:             uint(event.IntendedWaypointThroughputCalculationMode),
			ThresholdToCheckIfAirplaneIsCountedAtWaypointInNm:     event.ThresholdToCheckIfAirplaneIsCountedAtWaypointInNm,
			CalculationFallbackGroundSpeed:                        event.CalculationFallbackGroundSpeed,
			HighSimulationAccuracy:                                event.HighSimulationAccuracy,
		},
		Airports:                  airports,
		Waypoints:                 waypoints,
		RouteSegments:             routeSegments,
		Sectors:                   sectors,
		TagLimits:                 tagLimits,
		Slots:                     slots,
		DeferredDeparturePairIds:  convertedDeferredPairs,
		PreferredDeparturePairIds: convertedPreferredPairs,
	}
}

func fetchSimulatorData(id uint64) (*models.VATSIMEvent, *models.SlotRevision, error) {
	var event models.VATSIMEvent
	result := database.DB.
		Preload("Airports.Waypoint").
		Preload("RouteSegments").
		Preload("RouteSegments.Tags.TagRef").
		Preload("RouteSegments.Locations.Waypoint").
		Preload("RouteSegments.ProvidedFacilityProgression").
		First(&event, id)
	if result.Error != nil {
		return nil, nil, fiber.NewError(fiber.StatusNotFound, "event not found")
	}

	var revision models.SlotRevision
	q := database.DB.
		Preload("Slots").
		Preload("Slots.DepartureAirport.Waypoint").
		Preload("Slots.ArrivalAirport.Waypoint").
		Preload("Slots.RouteSegments").
		Preload("Slots.RouteSegments.Tags.TagRef").
		Preload("Slots.RouteSegments.Locations.Waypoint").
		Preload("Slots.RouteSegments.ProvidedFacilityProgression").
		Where("event_id = ? AND EXISTS (SELECT 1 FROM slots WHERE slot_revision_id = slot_revisions.id)", id).
		Order("number DESC").
		First(&revision)

	if q.Error != nil {
		return &event, nil, nil
	}
	sortSlotRouteSegments(revision.Slots)
	return &event, &revision, nil
}

func callSimulator(ctx context.Context, path string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.C.SimulatorURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, err
}

func airportWaypointLookup(tx *gorm.DB, eventID uint) map[int64]uint {
	var airports []models.Airport
	tx.Where("event_id = ?", eventID).Find(&airports)
	m := make(map[int64]uint, len(airports))
	for _, a := range airports {
		m[a.WaypointID] = a.ID
	}
	return m
}

func updateAirportDepartureTimeWindows(db *gorm.DB, resp simResponseEvent, airportByWaypoint map[int64]uint) {
	for _, a := range resp.Airports {
		if a.DepartureTimeWindowStart.IsZero() {
			continue
		}
		airportID := airportByWaypoint[a.Id]
		if airportID == 0 {
			continue
		}
		t := a.DepartureTimeWindowStart.Time
		db.Model(&models.Airport{}).Where("id = ?", airportID).Update("departure_time_window_start", t)
	}
}

func updateAirportEarliestArrivals(db *gorm.DB, resp simResponseEvent, airportByWaypoint map[int64]uint) {
	// Find the earliest non-zero ProjectedArrivalTime per arrival airport waypointID.
	earliest := make(map[int64]time.Time)
	for _, s := range resp.Slots {
		if s.ProjectedArrivalTime.IsZero() {
			continue
		}
		t := s.ProjectedArrivalTime.Time
		if prev, ok := earliest[s.ArrivalAirport]; !ok || t.Before(prev) {
			earliest[s.ArrivalAirport] = t
		}
	}
	for waypointID, t := range earliest {
		airportID := airportByWaypoint[waypointID]
		if airportID == 0 {
			continue
		}
		db.Model(&models.Airport{}).Where("id = ?", airportID).Update("earliest_arrival_time", t)
	}
}

func writeThroughputStates(tx *gorm.DB, revisionID uint, resp simResponseEvent, airportByWaypoint map[int64]uint) error {
	rows := make([]models.ThroughputState, 0, len(resp.Airports)+len(resp.Waypoints)+len(resp.RouteSegments)+len(resp.Sectors))

	for _, a := range resp.Airports {
		var ts *time.Time
		if !a.DepartureTimeWindowStart.IsZero() {
			t := a.DepartureTimeWindowStart.Time
			ts = &t
		}
		rows = append(rows, models.ThroughputState{
			SlotRevisionID:           revisionID,
			ThroughputPointType:      "airport",
			ThroughputPointID:        int64(airportByWaypoint[a.Id]),
			MaximumSlots:             a.MaximumSlots,
			SlotsAllocated:           a.SlotsAllocated,
			DepartureTimeWindowStart: ts,
		})
	}
	for _, w := range resp.Waypoints {
		rows = append(rows, models.ThroughputState{
			SlotRevisionID:      revisionID,
			ThroughputPointType: "waypoint",
			ThroughputPointID:   w.Id,
			MaximumSlots:        w.MaximumSlots,
			SlotsAllocated:      w.SlotsAllocated,
		})
	}
	for _, r := range resp.RouteSegments {
		rows = append(rows, models.ThroughputState{
			SlotRevisionID:      revisionID,
			ThroughputPointType: "route_segment",
			ThroughputPointID:   r.Id,
			MaximumSlots:        r.MaximumSlots,
			SlotsAllocated:      r.SlotsAllocated,
		})
	}
	for _, s := range resp.Sectors {
		rows = append(rows, models.ThroughputState{
			SlotRevisionID:      revisionID,
			ThroughputPointType: "sector",
			ThroughputPointID:   s.Id,
			MaximumSlots:        s.MaximumSlots,
			SlotsAllocated:      s.SlotsAllocated,
		})
	}

	if len(rows) == 0 {
		return nil
	}
	return tx.CreateInBatches(rows, 500).Error
}

var snapshotPreCopyDDL = []string{
	`DROP INDEX IF EXISTS idx_throughput_snapshots_slot_revision_id`,
	`ALTER TABLE throughput_snapshots DROP CONSTRAINT IF EXISTS fk_throughput_snapshots_slot`,
}

var snapshotPostCopyDDL = []string{
	`CREATE INDEX idx_throughput_snapshots_slot_revision_id ON throughput_snapshots (slot_revision_id)`,
	`ALTER TABLE throughput_snapshots ADD CONSTRAINT fk_throughput_snapshots_slot FOREIGN KEY (slot_id) REFERENCES slots(id) NOT VALID`,
}

var slotPositionPreCopyDDL = []string{
	`DROP INDEX IF EXISTS idx_slot_positions_slot_id`,
	`DROP INDEX IF EXISTS idx_slot_positions_timestamp`,
}

var slotPositionPostCopyDDL = []string{
	`CREATE INDEX idx_slot_positions_slot_id ON slot_positions (slot_id)`,
	`CREATE INDEX idx_slot_positions_timestamp ON slot_positions (timestamp)`,
}

func writeThroughputSnapshots(tx *gorm.DB, revisionID uint, resp simResponseEvent, airportByWaypoint map[int64]uint) error {
	// Pre-calculate total capacity to avoid reallocations
	total := 0
	for _, a := range resp.Airports {
		for _, ids := range a.SlotsFrames {
			total += len(ids)
		}
	}
	for _, w := range resp.Waypoints {
		for _, ids := range w.SlotsFrames {
			total += len(ids)
		}
	}
	for _, r := range resp.RouteSegments {
		for _, ids := range r.SlotsFrames {
			total += len(ids)
		}
	}
	for _, s := range resp.Sectors {
		for _, ids := range s.SlotsFrames {
			total += len(ids)
		}
	}

	if total == 0 {
		return nil
	}

	rows := make([]snapshotRow, 0, total)

	for _, a := range resp.Airports {
		pid := int64(airportByWaypoint[a.Id])
		for minuteOffset, slotIDs := range a.SlotsFrames {
			for _, slotID := range slotIDs {
				rows = append(rows, snapshotRow{revisionID, "airport", pid, minuteOffset, slotID})
			}
		}
	}
	for _, w := range resp.Waypoints {
		for minuteOffset, slotIDs := range w.SlotsFrames {
			for _, slotID := range slotIDs {
				rows = append(rows, snapshotRow{revisionID, "waypoint", w.Id, minuteOffset, slotID})
			}
		}
	}
	for _, r := range resp.RouteSegments {
		for minuteOffset, slotIDs := range r.SlotsFrames {
			for _, slotID := range slotIDs {
				rows = append(rows, snapshotRow{revisionID, "route_segment", r.Id, minuteOffset, slotID})
			}
		}
	}
	for _, s := range resp.Sectors {
		for minuteOffset, slotIDs := range s.SlotsFrames {
			for _, slotID := range slotIDs {
				rows = append(rows, snapshotRow{revisionID, "sector", s.Id, minuteOffset, slotID})
			}
		}
	}

	log.Info().Int("snapshotRows", len(rows)).Msg("[writeThroughputSnapshots] COPY inserting")

	// Extract the raw *sql.DB from GORM, then grab a pgx conn for COPY.
	sqlDB, err := tx.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("acquire sql.Conn: %w", err)
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		// The gorm postgres driver wraps pgx/v5/stdlib whose Conn type
		// exposes the underlying *pgx.Conn via a Conn() method.
		type pgxConner interface {
			Conn() *pgx.Conn
		}
		pgxConn := driverConn.(pgxConner).Conn()
		ctx := context.Background()

		// Drop indexes/FK before bulk load — rebuilding once after is far
		// cheaper than maintaining B-trees per-row during COPY.
		for _, ddl := range snapshotPreCopyDDL {
			if _, execErr := pgxConn.Exec(ctx, ddl); execErr != nil {
				log.Warn().Err(execErr).Str("ddl", ddl).Msg("[writeThroughputSnapshots] pre-copy DDL warning")
			}
		}

		_, copyErr := pgxConn.CopyFrom(
			ctx,
			pgx.Identifier{"throughput_snapshots"},
			[]string{"slot_revision_id", "throughput_point_type", "throughput_point_id", "minute_offset", "slot_id"},
			pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
				r := &rows[i]
				return []any{r.revisionID, r.pointType, r.pointID, r.minute, r.slotID}, nil
			}),
		)

		// Recreate only the slot_revision_id index (the only one read queries use).
		for _, ddl := range snapshotPostCopyDDL {
			if _, execErr := pgxConn.Exec(ctx, ddl); execErr != nil {
				log.Error().Err(execErr).Str("ddl", ddl).Msg("[writeThroughputSnapshots] post-copy DDL failed")
			}
		}

		return copyErr
	})
}

func writeSlotPositions(tx *gorm.DB, resp simResponseEvent) error {
	var rows []positionRow
	for _, s := range resp.Slots {
		if s.Id == 0 || len(s.SimulatedPositions) == 0 {
			continue
		}
		for tsStr, coords := range s.SimulatedPositions {
			ts, err := time.Parse(time.RFC3339, tsStr)
			if err != nil {
				continue
			}
			if len(coords) < 2 {
				continue
			}
			rows = append(rows, positionRow{
				slotID:    s.Id,
				timestamp: ts,
				latitude:  coords[0],
				longitude: coords[1],
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}

	log.Info().Int("positionRows", len(rows)).Msg("[writeSlotPositions] COPY inserting")

	sqlDB, err := tx.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("acquire sql.Conn: %w", err)
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		type pgxConner interface {
			Conn() *pgx.Conn
		}
		pgxConn := driverConn.(pgxConner).Conn()
		ctx := context.Background()

		for _, ddl := range slotPositionPreCopyDDL {
			if _, execErr := pgxConn.Exec(ctx, ddl); execErr != nil {
				log.Warn().Err(execErr).Str("ddl", ddl).Msg("[writeSlotPositions] pre-copy DDL warning")
			}
		}

		_, copyErr := pgxConn.CopyFrom(
			ctx,
			pgx.Identifier{"slot_positions"},
			[]string{"slot_id", "timestamp", "latitude", "longitude"},
			pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
				r := &rows[i]
				return []any{r.slotID, r.timestamp, r.latitude, r.longitude}, nil
			}),
		)

		for _, ddl := range slotPositionPostCopyDDL {
			if _, execErr := pgxConn.Exec(ctx, ddl); execErr != nil {
				log.Error().Err(execErr).Str("ddl", ddl).Msg("[writeSlotPositions] post-copy DDL failed")
			}
		}

		return copyErr
	})
}

func saveCalculationResult(eventID uint, resp simResponseEvent, commentary string) (uint, uint, error) {
	var revisionID uint
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var maxNumber uint
		tx.Model(&models.SlotRevision{}).Where("event_id = ?", eventID).Select("COALESCE(MAX(number), 0)").Scan(&maxNumber)

		revision := models.SlotRevision{
			EventID:                        eventID,
			Number:                         maxNumber + 1,
			SlotGenerationOutputCommentary: commentary,
		}
		if err := tx.Create(&revision).Error; err != nil {
			return err
		}
		revisionID = revision.ID

		airportByWaypoint := airportWaypointLookup(tx, eventID)

		for _, s := range resp.Slots {
			slot := models.Slot{
				SlotRevisionID:       revision.ID,
				DepartureTime:        s.DepartureTime.Time,
				ProjectedArrivalTime: s.ProjectedArrivalTime.Time,
				DepartureAirportID:   airportByWaypoint[s.DepartureAirport],
				ArrivalAirportID:     airportByWaypoint[s.ArrivalAirport],
			}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
			for i, rsID := range s.RouteSegments {
				if err := tx.Create(&models.SlotRouteSegment{
					SlotID:         slot.ID,
					RouteSegmentID: rsID,
					Order:          uint(i),
				}).Error; err != nil {
					return err
				}
			}
		}

		return writeThroughputStates(tx, revision.ID, resp, airportByWaypoint)
	})
	if err == nil {
		airportByWaypoint := airportWaypointLookup(database.DB, eventID)
		updateAirportDepartureTimeWindows(database.DB, resp, airportByWaypoint)
		updateAirportEarliestArrivals(database.DB, resp, airportByWaypoint)
	}
	return revisionID, 0, err
}

func saveSimulationResult(eventID uint, resp simResponseEvent, commentary string, _ json.RawMessage, _ json.RawMessage) (uint, uint, error) {
	log.Info().
		Uint("eventId", eventID).
		Int("slotsInResponse", len(resp.Slots)).
		Msg("[saveSimulation] starting")

	var revisionID uint
	var revisionNumber uint

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Get previous revision's draft entries before we create the new one
		var prevRevision models.SlotRevision
		var draftEntries []models.SlotDraftEntry
		if err := tx.Where("event_id = ?", eventID).Order("number DESC").First(&prevRevision).Error; err == nil {
			tx.Where("slot_revision_id = ?", prevRevision.ID).Find(&draftEntries)
		}

		// Create NEW revision with simulated slots
		var maxNumber uint
		tx.Model(&models.SlotRevision{}).Where("event_id = ?", eventID).Select("COALESCE(MAX(number), 0)").Scan(&maxNumber)

		revision := models.SlotRevision{
			EventID:                        eventID,
			Number:                         maxNumber + 1,
			SlotGenerationOutputCommentary: commentary,
		}
		if err := tx.Create(&revision).Error; err != nil {
			return err
		}
		revisionID = revision.ID
		revisionNumber = revision.Number
		log.Info().
			Uint("revisionId", revision.ID).
			Uint("revisionNumber", revision.Number).
			Msg("[saveSimulation] created new revision")

		airportByWaypoint := airportWaypointLookup(tx, eventID)

		// Create slots on the new revision
		for _, s := range resp.Slots {
			slot := models.Slot{
				SlotRevisionID:       revision.ID,
				DepartureTime:        s.DepartureTime.Time,
				ProjectedArrivalTime: s.ProjectedArrivalTime.Time,
				DepartureAirportID:   airportByWaypoint[s.DepartureAirport],
				ArrivalAirportID:     airportByWaypoint[s.ArrivalAirport],
			}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
			for i, rsID := range s.RouteSegments {
				if err := tx.Create(&models.SlotRouteSegment{
					SlotID:         slot.ID,
					RouteSegmentID: rsID,
					Order:          uint(i),
				}).Error; err != nil {
					return err
				}
			}
		}

		// Copy draft entries from previous revision to this new revision
		if len(draftEntries) > 0 {
			for i := range draftEntries {
				draftEntries[i].ID = 0
				draftEntries[i].SlotRevisionID = revision.ID
			}
			if err := tx.Create(&draftEntries).Error; err != nil {
				log.Error().Err(err).Msg("[saveSimulation] failed to copy draft entries")
			} else {
				log.Info().Int("copied", len(draftEntries)).Msg("[saveSimulation] copied draft entries to new revision")
			}
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Uint("eventId", eventID).Msg("[saveSimulation] transaction failed")
		return revisionID, 0, err
	}
	log.Info().Uint("revisionId", revisionID).Msg("[saveSimulation] transaction committed")

	// ── Background: write throughput data, clean up old revisions ─────────────
	// Snapshots/states for old revisions are purged here, keeping only the newest.
	// resp is captured by value; revisionID/eventID are primitives — safe for goroutine.
	go func(rid uint, eid uint, simResp simResponseEvent) {
		log.Info().Uint("revisionId", rid).Uint("eventId", eid).Msg("[saveSimulation] background throughput write starting")
		bgStart := time.Now()

		// Drop all throughput data for every revision of this event except the current one.
		t := time.Now()
		database.DB.Exec(
			"DELETE FROM throughput_snapshots WHERE slot_revision_id IN (SELECT id FROM slot_revisions WHERE event_id = ? AND id != ?)",
			eid, rid,
		)
		database.DB.Exec(
			"DELETE FROM throughput_states WHERE slot_revision_id IN (SELECT id FROM slot_revisions WHERE event_id = ? AND id != ?)",
			eid, rid,
		)
		// Also replace any existing data for the current revision (handles re-runs).
		database.DB.Where("slot_revision_id = ?", rid).Delete(&models.ThroughputSnapshot{})
		database.DB.Where("slot_revision_id = ?", rid).Delete(&models.ThroughputState{})
		database.DB.Exec("DELETE FROM slot_positions WHERE slot_id IN (SELECT id FROM slots WHERE slot_revision_id = ?)", rid)
		log.Info().Dur("elapsed", time.Since(t)).Msg("[saveSimulation] old data purged")

		airportByWaypoint := airportWaypointLookup(database.DB, eid)

		t = time.Now()
		writeThroughputStates(database.DB, rid, simResp, airportByWaypoint)
		updateAirportDepartureTimeWindows(database.DB, simResp, airportByWaypoint)
		updateAirportEarliestArrivals(database.DB, simResp, airportByWaypoint)
		log.Info().Dur("elapsed", time.Since(t)).Msg("[saveSimulation] writeThroughputStates + airport time windows done")

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			t := time.Now()
			writeThroughputSnapshots(database.DB, rid, simResp, airportByWaypoint)
			log.Info().Dur("elapsed", time.Since(t)).Msg("[saveSimulation] writeThroughputSnapshots done")
		}()
		go func() {
			defer wg.Done()
			t := time.Now()
			writeSlotPositions(database.DB, simResp)
			log.Info().Dur("elapsed", time.Since(t)).Msg("[saveSimulation] writeSlotPositions done")
		}()
		wg.Wait()

		log.Info().Uint("revisionId", rid).Dur("totalElapsed", time.Since(bgStart)).Msg("[saveSimulation] background throughput write complete")
		simStatusStore.Delete(uint64(eid))
	}(revisionID, eventID, resp)

	return revisionID, revisionNumber, nil
}

func invokeSimulator(c fiber.Ctx, path string, includeSlots bool, save func(uint, simResponseEvent, string) (uint, uint, error)) error {
	if config.C.SimulatorURL == "" {
		return fiber.NewError(fiber.StatusServiceUnavailable, "simulator not configured")
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	event, revision, err := fetchSimulatorData(id)
	if err != nil {
		return err
	}

	slotCount := 0
	if revision != nil {
		slotCount = len(revision.Slots)
	}
	log.Info().
		Uint64("eventId", id).
		Int("slots", slotCount).
		Str("path", path).
		Msg("[invokeSimulator] fetched data, calling simulator")

	payload := buildSimEvent(*event, revision, includeSlots)

	body, err := json.Marshal(payload)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to serialize payload")
	}

	ctx, cancel := context.WithTimeout(c.Context(), simulatorTimeout)
	defer cancel()

	simStatusStore.Store(id, "running")
	respBody, statusCode, err := callSimulator(ctx, path, body)
	if err != nil {
		simStatusStore.Delete(id)
		log.Error().Err(err).Str("path", path).Msg("[invokeSimulator] simulator call failed")
		return fiber.NewError(fiber.StatusBadGateway, "simulator request failed: "+err.Error())
	}
	log.Info().
		Str("path", path).
		Int("status", statusCode).
		Int("responseBytes", len(respBody)).
		Msg("[invokeSimulator] simulator responded")
	if statusCode == http.StatusInternalServerError {
		simStatusStore.Delete(id)
		return fiber.NewError(fiber.StatusBadGateway, "simulator calculation error: "+string(respBody))
	}
	if statusCode != http.StatusOK {
		simStatusStore.Delete(id)
		return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("simulator returned %d", statusCode))
	}

	// Strip a UTF-8 BOM (\xEF\xBB\xBF) if the simulator includes one.
	respBody = bytes.TrimPrefix(respBody, []byte("\xef\xbb\xbf"))

	var simResp simResponseEvent
	if err := json.Unmarshal(respBody, &simResp); err != nil {
		simStatusStore.Delete(id)
		// Log the first 512 bytes of the raw body to help diagnose encoding issues.
		preview := respBody
		if len(preview) > 512 {
			preview = preview[:512]
		}
		log.Error().
			Str("path", path).
			Int("statusCode", statusCode).
			Str("bodyPreview", string(preview)).
			Msgf("failed to parse simulator response: %s", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to parse simulator response: "+err.Error())
	}

	log.Info().
		Str("path", path).
		Int("slotsInResponse", len(simResp.Slots)).
		Int("airportsInResponse", len(simResp.Airports)).
		Msg("[invokeSimulator] parsed simulator response, calling save")

	// Persist raw response so it can be retrieved via the latest-response endpoint.
	simLatestResponseStore.Store(id, respBody)

	simStatusStore.Store(id, "sim_responded")

	commentary := strings.Join(simResp.CalculationParameters.SlotGenerationOutputComments, "\n")
	if path == "/simulateEvent" {
		commentary = strings.Join(simResp.CalculationParameters.SimulationOutputComments, "\n")
	}

	revisionID, draftRevisionNumber, err := save(uint(id), simResp, commentary)
	if err != nil {
		simStatusStore.Delete(id)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save simulator results: "+err.Error())
	}

	// Status "saved" — background goroutine will clear it once throughput data is written.
	simStatusStore.Store(id, "saved")

	resp := fiber.Map{
		"slotRevisionId":                 revisionID,
		"slotRevisionNumber":             draftRevisionNumber,
		"slotGenerationOutputCommentary": strings.Join(simResp.CalculationParameters.SlotGenerationOutputComments, "\n"),
		"simulationOutputCommentary":     strings.Join(simResp.CalculationParameters.SimulationOutputComments, "\n"),
	}
	return c.JSON(resp)
}

// PreviewCalculatePayload godoc
//
//	@Summary	Preview the payload that would be sent to /createSlotDistribution
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int		true	"Event ID"
//	@Success	200	{object}	object
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id}/calculate-slots/preview [get]
func PreviewCalculatePayload(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}
	event, revision, err := fetchSimulatorData(id)
	if err != nil {
		return err
	}
	return c.JSON(buildSimEvent(*event, revision, false))
}

// PreviewSimulatePayload godoc
//
//	@Summary	Preview the payload that would be sent to /simulateEvent (includes slots)
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int		true	"Event ID"
//	@Success	200	{object}	object
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/events/{id}/simulate-slots/preview [get]
func PreviewSimulatePayload(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}
	event, revision, err := fetchSimulatorData(id)
	if err != nil {
		return err
	}
	return c.JSON(buildSimEvent(*event, revision, true))
}

// CalculateSlots godoc
//
//	@Summary	Send event data to simulator to calculate slot distribution
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int		true	"Event ID"
//	@Success	200	{object}	object	"{ slotRevisionId: uint }"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Failure	502	{object}	models.ErrorResponse
//	@Failure	503	{object}	models.ErrorResponse
//	@Router		/events/{id}/calculate-slots [post]
func CalculateSlots(c fiber.Ctx) error {
	return invokeSimulator(c, "/createSlotDistribution", false, saveCalculationResult)
}

// SimulateSlots godoc
//
//	@Summary	Send event data and current slots to simulator for simulation
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int		true	"Event ID"
//	@Success	200	{object}	object	"{ slotRevisionId: uint, draftRevisionNumber?: uint }"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Failure	502	{object}	models.ErrorResponse
//	@Failure	503	{object}	models.ErrorResponse
//	@Router		/events/{id}/simulate-slots [post]
func SimulateSlots(c fiber.Ctx) error {
	return invokeSimulator(c, "/simulateEvent", true, func(eventID uint, resp simResponseEvent, commentary string) (uint, uint, error) {
		return saveSimulationResult(eventID, resp, commentary, json.RawMessage(nil), json.RawMessage(nil))
	})
}
