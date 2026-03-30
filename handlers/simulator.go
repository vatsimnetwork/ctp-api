package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"github.com/vatsimnetwork/ctp-api/config"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

const simulatorTimeout = 2 * time.Minute

type simCalculationParameters struct {
	RecalculateMaximumAirportSlots                        bool    `json:"recalculateMaximumAirportSlots"`
	IntendedSlotGenerationMode                            uint    `json:"intendedSlotGenerationMode"`
	DepartureTimeWindowOffsetSynchronizationLongitude     float64 `json:"departureTimeWindowOffsetSynchronizationLongitude"`
	SimulationAnalysisResolutionInMinutes                 uint    `json:"simulationAnalysisResolutionInMinutes"`
	ShouldSimulationUseActualWeatherForecastData          bool    `json:"shouldSimulationUseActualWeatherForecastData"`
	IntendedDepartureTimeWindowOffsetsCalculationMode     uint    `json:"intendedDepartureTimeWindowOffsetsCalculationMode"`
	DepartureTimeWindowOffsetSynchronizationTimeOfDay     string  `json:"departureTimeWindowOffsetSynchronizationTimeOfDay"`
	CalculateThroughputDataOnlyForManuallyProvidedSectors bool    `json:"calculateThroughputDataOnlyForManuallyProvidedSectors"`
	IntendedWaypointThroughputCalculationMode             uint    `json:"intendedWaypointThroughputCalculationMode"`
	ThresholdToCheckIfAirplaneIsCountedAtWaypointInNm     float64 `json:"thresholdToCheckIfAirplaneIsCountedAtWaypointInNm"`
	CalculationFallbackGroundSpeed                        float64 `json:"calculationFallbackGroundSpeed"`
	HighSimulationAccuracy                                bool    `json:"highSimulationAccuracy"`
}

type simWaypoint struct {
	Id                     int64   `json:"id"`
	Identifier             string  `json:"identifier"`
	MaximumAircraftPerHour uint16  `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16  `json:"maximumSlots"`
	Latitude               float64 `json:"latitude"`
	Longitude              float64 `json:"longitude"`
}

type simAirport struct {
	Id                     int64   `json:"id"`
	Identifier             string  `json:"identifier"`
	MaximumAircraftPerHour uint16  `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16  `json:"maximumSlots"`
	Latitude               float64 `json:"latitude"`
	Longitude              float64 `json:"longitude"`
	NumberOfVotes          uint16  `json:"numberOfVotes"`
}

type simSector struct {
	Id                     uint   `json:"id"`
	Identifier             string `json:"identifier"`
	MaximumAircraftPerHour uint16 `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16 `json:"maximumSlots"`
}

type simRouteSegment struct {
	Id                          uint     `json:"id"`
	Identifier                  string   `json:"identifier"`
	MaximumAircraftPerHour      uint16   `json:"maximumAircraftPerHour"`
	MaximumSlots                uint16   `json:"maximumSlots"`
	RouteString                 string   `json:"routeString"`
	RouteSegmentGroup           string   `json:"routeSegmentGroup"`
	Color                       string   `json:"color"`
	Enabled                     bool     `json:"enabled"`
	RouteSegmentTags            []string `json:"routeSegmentTags"`
	ProvidedFacilityProgression []uint   `json:"providedFacilityProgression"`
	Locations                   []int64  `json:"locations"`
	RouteRevision               uint     `json:"routeRevision"`
}

type simSlot struct {
	Id                   uint   `json:"id"`
	DepartureTime        string `json:"departureTime"`
	ProjectedArrivalTime string `json:"projectedArrivalTime"`
	DepartureAirport     int64  `json:"departureAirport"` // airport WaypointID
	ArrivalAirport       int64  `json:"arrivalAirport"`   // airport WaypointID
	RouteSegments        []uint `json:"routeSegments"`    // route segment IDs
}

type simEvent struct {
	Id                    uint                     `json:"id"`
	Title                 string                   `json:"title"`
	RouteRevision         uint                     `json:"routeRevision"`
	SlotRevision          uint                     `json:"slotRevision"`
	Date                  string                   `json:"date"`
	DepartureTimeWindow   string                   `json:"departureTimeWindow"` // "hh:mm:ss" (C# TimeSpan)
	CalculationParameters simCalculationParameters `json:"calculationParameters"`
	Airports              []simAirport             `json:"airports"`
	Waypoints             []simWaypoint            `json:"waypoints"`
	RouteSegments         []simRouteSegment        `json:"routeSegments"`
	Sectors               []simSector              `json:"sectors"`
	Slots                 []simSlot                `json:"slots"`
}

// simTime is a time.Time that tolerates the simulator's non-RFC3339 timestamps
// (e.g. "2026-04-25T06:00:00" without timezone) and treats the C# zero-value
// "0001-01-01T00:00:00" as a Go zero time (IsZero() == true).
type simTime struct {
	time.Time
}

func (t *simTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.9999999",
		"2006-01-02T15:04:05",
	} {
		if parsed, err := time.Parse(layout, s); err == nil {
			if parsed.Year() < 100 {
				t.Time = time.Time{}
			} else {
				t.Time = parsed
			}
			return nil
		}
	}
	t.Time = time.Time{}
	return nil
}

type simResponseThroughput struct {
	Id             int64          `json:"id"`
	MaximumSlots   uint16         `json:"maximumSlots"`
	SlotsAllocated uint16         `json:"slotsAllocated"`
	SlotsFrames    map[int][]uint `json:"slotsAnalysisFramesViaMinutesFromSynchronizationTime"`
}

type simResponseAirport struct {
	simResponseThroughput
	DepartureTimeWindowStart simTime `json:"departureTimeWindowStart"`
}

type simResponseSlot struct {
	Id                   uint    `json:"id"`
	DepartureTime        simTime `json:"departureTime"`
	ProjectedArrivalTime simTime `json:"projectedArrivalTime"`
	DepartureAirport     int64   `json:"departureAirport"`
	ArrivalAirport       int64   `json:"arrivalAirport"`
	RouteSegments        []uint  `json:"routeSegments"`
}

type simResponseCalcParams struct {
	SlotGenerationOutputCommentary string `json:"slotGenerationOutputCommentary"`
	SimulationOutputCommentary     string `json:"simulationOutputCommentary"`
}

type simResponseEvent struct {
	CalculationParameters simResponseCalcParams   `json:"calculationParameters"`
	Airports              []simResponseAirport    `json:"airports"`
	Waypoints             []simResponseThroughput `json:"waypoints"`
	RouteSegments         []simResponseThroughput `json:"routeSegments"`
	Slots                 []simResponseSlot       `json:"slots"`
}

func mapAirport(a models.Airport, departureHours float64) simAirport {
	if departureHours <= 0 {
		departureHours = 1
	}
	maxPerHour := uint16(math.Ceil(float64(a.MaximumSlots) / departureHours))
	return simAirport{
		Id:                     a.WaypointID,
		Identifier:             a.Waypoint.Identifier,
		MaximumAircraftPerHour: maxPerHour,
		MaximumSlots:           a.MaximumSlots,
		Latitude:               a.Waypoint.Latitude,
		Longitude:              a.Waypoint.Longitude,
		NumberOfVotes:          a.NumberOfVotes,
	}
}

func mapRouteSegment(r models.RouteSegment, airportWaypointIDs map[int64]bool) simRouteSegment {
	tags := make([]string, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, t.Tag)
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

	return simRouteSegment{
		Id:                          r.ID,
		Identifier:                  r.Identifier,
		MaximumAircraftPerHour:      r.MaximumAircraftPerHour,
		RouteString:                 r.RouteString,
		RouteSegmentGroup:           r.RouteSegmentGroup,
		Color:                       r.Color,
		Enabled:                     r.Enabled,
		RouteSegmentTags:            tags,
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
	for _, a := range event.Airports {
		airports = append(airports, mapAirport(a, departureHours))
		airportWaypointIDs[a.WaypointID] = true
	}

	waypointByID := make(map[int64]simWaypoint)
	for _, r := range event.RouteSegments {
		for _, l := range r.Locations {
			if airportWaypointIDs[l.WaypointID] {
				continue
			}
			if _, exists := waypointByID[l.WaypointID]; !exists {
				waypointByID[l.WaypointID] = simWaypoint{
					Id:                     l.WaypointID,
					Identifier:             l.Waypoint.Identifier,
					MaximumAircraftPerHour: l.Waypoint.MaximumAircraftPerHour,
					MaximumSlots:           l.Waypoint.MaximumSlots,
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
		routeSegments = append(routeSegments, mapRouteSegment(r, airportWaypointIDs))
	}

	sectorByID := make(map[uint]simSector)
	for _, r := range event.RouteSegments {
		for _, s := range r.ProvidedFacilityProgression {
			if _, exists := sectorByID[s.ID]; !exists {
				sectorByID[s.ID] = simSector{
					Id:                     s.ID,
					Identifier:             s.Identifier,
					MaximumAircraftPerHour: s.MaximumAircraftPerHour,
				}
			}
		}
	}
	sectors := make([]simSector, 0, len(sectorByID))
	for _, s := range sectorByID {
		sectors = append(sectors, s)
	}
	sort.Slice(sectors, func(i, j int) bool { return sectors[i].Id < sectors[j].Id })

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

	return simEvent{
		Id:                  event.ID,
		Title:               event.Title,
		RouteRevision:       event.RouteRevision,
		SlotRevision:        slotRevisionNumber,
		Date:                event.Date.Format("2006-01-02"),
		DepartureTimeWindow: formatDepartureTimeWindow(event.DepartureTimeWindow),
		CalculationParameters: simCalculationParameters{
			RecalculateMaximumAirportSlots:                        event.RecalculateMaximumAirportSlots,
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
		Airports:      airports,
		Waypoints:     waypoints,
		RouteSegments: routeSegments,
		Sectors:       sectors,
		Slots:         slots,
	}
}

func fetchSimulatorData(id uint64) (*models.VATSIMEvent, *models.SlotRevision, error) {
	var event models.VATSIMEvent
	result := database.DB.
		Preload("Airports.Waypoint").
		Preload("RouteSegments").
		Preload("RouteSegments.Tags").
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
		Preload("Slots.RouteSegments.Tags").
		Preload("Slots.RouteSegments.Locations.Waypoint").
		Preload("Slots.RouteSegments.ProvidedFacilityProgression").
		Where("event_id = ?", id).
		Order("number DESC").
		First(&revision)

	if q.Error != nil {
		return &event, nil, nil
	}
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

func writeThroughputStates(tx *gorm.DB, revisionID uint, resp simResponseEvent, airportByWaypoint map[int64]uint) error {
	for _, a := range resp.Airports {
		var ts *time.Time
		if !a.DepartureTimeWindowStart.IsZero() {
			t := a.DepartureTimeWindowStart.Time
			ts = &t
		}
		if err := tx.Create(&models.ThroughputState{
			SlotRevisionID:           revisionID,
			ThroughputPointType:      "airport",
			ThroughputPointID:        int64(airportByWaypoint[a.Id]),
			MaximumSlots:             a.MaximumSlots,
			SlotsAllocated:           a.SlotsAllocated,
			DepartureTimeWindowStart: ts,
		}).Error; err != nil {
			return err
		}
	}
	for _, w := range resp.Waypoints {
		if err := tx.Create(&models.ThroughputState{
			SlotRevisionID:      revisionID,
			ThroughputPointType: "waypoint",
			ThroughputPointID:   w.Id,
			MaximumSlots:        w.MaximumSlots,
			SlotsAllocated:      w.SlotsAllocated,
		}).Error; err != nil {
			return err
		}
	}
	for _, r := range resp.RouteSegments {
		if err := tx.Create(&models.ThroughputState{
			SlotRevisionID:      revisionID,
			ThroughputPointType: "route_segment",
			ThroughputPointID:   r.Id,
			MaximumSlots:        r.MaximumSlots,
			SlotsAllocated:      r.SlotsAllocated,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func writeThroughputSnapshots(tx *gorm.DB, revisionID uint, resp simResponseEvent, airportByWaypoint map[int64]uint) error {
	write := func(pointType string, pointID int64, frames map[int][]uint) error {
		for minuteOffset, slotIDs := range frames {
			for _, slotID := range slotIDs {
				if err := tx.Create(&models.ThroughputSnapshot{
					SlotRevisionID:      revisionID,
					ThroughputPointType: pointType,
					ThroughputPointID:   pointID,
					MinuteOffset:        minuteOffset,
					SlotID:              slotID,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}

	for _, a := range resp.Airports {
		if err := write("airport", int64(airportByWaypoint[a.Id]), a.SlotsFrames); err != nil {
			return err
		}
	}
	for _, w := range resp.Waypoints {
		if err := write("waypoint", w.Id, w.SlotsFrames); err != nil {
			return err
		}
	}
	for _, r := range resp.RouteSegments {
		if err := write("route_segment", r.Id, r.SlotsFrames); err != nil {
			return err
		}
	}
	return nil
}

func saveCalculationResult(eventID uint, resp simResponseEvent, commentary string) (uint, error) {
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
			if len(s.RouteSegments) > 0 {
				rsegs := make([]models.RouteSegment, 0, len(s.RouteSegments))
				for _, rsID := range s.RouteSegments {
					rsegs = append(rsegs, models.RouteSegment{ThroughputPoint: models.ThroughputPoint{ID: rsID}})
				}
				if err := tx.Model(&slot).Association("RouteSegments").Append(rsegs); err != nil {
					return err
				}
			}
		}

		return writeThroughputStates(tx, revision.ID, resp, airportByWaypoint)
	})
	return revisionID, err
}

func saveSimulationResult(eventID uint, resp simResponseEvent, commentary string) (uint, error) {
	var revisionID uint
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var revision models.SlotRevision
		if err := tx.Where("event_id = ?", eventID).Order("number DESC").First(&revision).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "no slot revision found to attach simulation results to")
		}
		revisionID = revision.ID

		if err := tx.Model(&revision).Update("simulation_output_commentary", commentary).Error; err != nil {
			return err
		}

		airportByWaypoint := airportWaypointLookup(tx, eventID)

		// Update departure/arrival times on each slot using the IDs the simulator echoes back.
		for _, s := range resp.Slots {
			if s.Id == 0 {
				continue
			}
			updates := map[string]interface{}{
				"departure_time":         s.DepartureTime.Time,
				"projected_arrival_time": s.ProjectedArrivalTime.Time,
			}
			tx.Model(&models.Slot{}).
				Where("id = ? AND slot_revision_id = ?", s.Id, revision.ID).
				Updates(updates)
		}

		tx.Where("slot_revision_id = ?", revision.ID).Delete(&models.ThroughputState{})
		tx.Where("slot_revision_id = ?", revision.ID).Delete(&models.ThroughputSnapshot{})

		if err := writeThroughputStates(tx, revision.ID, resp, airportByWaypoint); err != nil {
			return err
		}
		return writeThroughputSnapshots(tx, revision.ID, resp, airportByWaypoint)
	})
	return revisionID, err
}

func invokeSimulator(c fiber.Ctx, path string, includeSlots bool, save func(uint, simResponseEvent, string) (uint, error)) error {
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

	payload := buildSimEvent(*event, revision, includeSlots)

	body, err := json.Marshal(payload)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to serialize payload")
	}

	ctx, cancel := context.WithTimeout(c.Context(), simulatorTimeout)
	defer cancel()

	respBody, statusCode, err := callSimulator(ctx, path, body)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "simulator request failed: "+err.Error())
	}
	if statusCode == http.StatusInternalServerError {
		return fiber.NewError(fiber.StatusBadGateway, "simulator calculation error: "+string(respBody))
	}
	if statusCode != http.StatusOK {
		return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("simulator returned %d", statusCode))
	}

	// Strip a UTF-8 BOM (\xEF\xBB\xBF) if the simulator includes one.
	respBody = bytes.TrimPrefix(respBody, []byte("\xef\xbb\xbf"))

	var simResp simResponseEvent
	if err := json.Unmarshal(respBody, &simResp); err != nil {
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

	commentary := simResp.CalculationParameters.SlotGenerationOutputCommentary
	if path == "/simulateEvent" {
		commentary = simResp.CalculationParameters.SimulationOutputCommentary
	}

	revisionID, err := save(uint(id), simResp, commentary)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save simulator results: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"slotRevisionId":                 revisionID,
		"slotGenerationOutputCommentary": simResp.CalculationParameters.SlotGenerationOutputCommentary,
		"simulationOutputCommentary":     simResp.CalculationParameters.SimulationOutputCommentary,
	})
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
//	@Success	200	{object}	object	"{ slotRevisionId: uint }"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Failure	502	{object}	models.ErrorResponse
//	@Failure	503	{object}	models.ErrorResponse
//	@Router		/events/{id}/simulate-slots [post]
func SimulateSlots(c fiber.Ctx) error {
	return invokeSimulator(c, "/simulateEvent", true, saveSimulationResult)
}
