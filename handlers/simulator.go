package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/config"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
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
	Id                          uint       `json:"id"`
	Identifier                  string     `json:"identifier"`
	MaximumAircraftPerHour      uint16     `json:"maximumAircraftPerHour"`
	MaximumSlots                uint16     `json:"maximumSlots"`
	RouteString                 string     `json:"routeString"`
	RouteSegmentGroup           string     `json:"routeSegmentGroup"`
	Color                       string     `json:"color"`
	Enabled                     bool       `json:"enabled"`
	RouteSegmentTags            []string   `json:"routeSegmentTags"`
	ProvidedFacilityProgression []simSector `json:"providedFacilityProgression"`
	Locations                   []int64    `json:"locations"`
	RouteRevision               uint       `json:"routeRevision"`
}

type simSlot struct {
	Id                   uint              `json:"id"`
	DepartureTime        string            `json:"departureTime"`
	ProjectedArrivalTime string            `json:"projectedArrivalTime"`
	DepartureAirport     simAirport        `json:"departureAirport"`
	ArrivalAirport       simAirport        `json:"arrivalAirport"`
	RouteSegments        []simRouteSegment `json:"routeSegments"`
}

type simEvent struct {
	Id                    uint                     `json:"id"`
	Title                 string                   `json:"title"`
	RouteRevision         uint                     `json:"routeRevision"`
	SlotRevision          uint                     `json:"slotRevision"`
	Date                  string                   `json:"date"`
	DepartureTimeWindow   float64                  `json:"departureTimeWindow"`
	CalculationParameters simCalculationParameters `json:"calculationParameters"`
	Airports              []simAirport             `json:"airports"`
	Waypoints             []simWaypoint            `json:"waypoints"`
	RouteSegments         []simRouteSegment        `json:"routeSegments"`
	Slots                 []simSlot                `json:"slots"`
}

func mapAirport(a models.Airport) simAirport {
	return simAirport{
		Id:                     a.WaypointID,
		Identifier:             a.Waypoint.Identifier,
		MaximumAircraftPerHour: a.MaximumAircraftPerHour,
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

	pfp := make([]simSector, 0, len(r.ProvidedFacilityProgression))
	for _, s := range r.ProvidedFacilityProgression {
		pfp = append(pfp, simSector{
			Id:                     s.ID,
			Identifier:             s.Identifier,
			MaximumAircraftPerHour: s.MaximumAircraftPerHour,
		})
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

func mapSlot(s models.Slot, airportWaypointIDs map[int64]bool) simSlot {
	segs := make([]simRouteSegment, 0, len(s.RouteSegments))
	for _, r := range s.RouteSegments {
		segs = append(segs, mapRouteSegment(r, airportWaypointIDs))
	}
	return simSlot{
		Id:                   s.ID,
		DepartureTime:        s.DepartureTime.UTC().Format(time.RFC3339),
		ProjectedArrivalTime: s.ProjectedArrivalTime.UTC().Format(time.RFC3339),
		DepartureAirport:     mapAirport(s.DepartureAirport),
		ArrivalAirport:       mapAirport(s.ArrivalAirport),
		RouteSegments:        segs,
	}
}

func normalizeTimeOfDay(s string) string {
	if strings.Count(s, ":") == 1 {
		return s + ":00"
	}
	return s
}

func buildSimEvent(event models.VATSIMEvent, revision *models.SlotRevision, includeSlots bool) simEvent {
	airports := make([]simAirport, 0, len(event.Airports))
	airportWaypointIDs := make(map[int64]bool, len(event.Airports))
	for _, a := range event.Airports {
		airports = append(airports, mapAirport(a))
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

	slots := []simSlot{}
	var slotRevisionNumber uint
	if revision != nil {
		slotRevisionNumber = revision.Number
		if includeSlots {
			for _, s := range revision.Slots {
				slots = append(slots, mapSlot(s, airportWaypointIDs))
			}
		}
	}

	return simEvent{
		Id:                  event.ID,
		Title:               event.Title,
		RouteRevision:       event.RouteRevision,
		SlotRevision:        slotRevisionNumber,
		Date:                event.Date.Format("2006-01-02"),
		DepartureTimeWindow: time.Duration(event.DepartureTimeWindow).Seconds(),
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

func forwardToSimulator(c fiber.Ctx, path string, includeSlots bool) error {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.C.SimulatorURL+path, bytes.NewReader(body))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to build simulator request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "simulator request failed: "+err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to read simulator response")
	}

	c.Set("Content-Type", "application/json")
	return c.Status(resp.StatusCode).Send(respBody)
}

// CalculateSlots godoc
//
//	@Summary	Send event data to simulator to calculate optimal slots
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int		true	"Event ID"
//	@Success	200	{object}	object	"Simulator response with calculated slots"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Failure	502	{object}	models.ErrorResponse
//	@Failure	503	{object}	models.ErrorResponse
//	@Router		/events/{id}/calculate-slots [post]
func CalculateSlots(c fiber.Ctx) error {
	return forwardToSimulator(c, "/calculate-slots", false)
}

// SimulateSlots godoc
//
//	@Summary	Send event data and current slots to simulator for simulation
//	@Tags		simulator
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int		true	"Event ID"
//	@Success	200	{object}	object	"Simulator response with simulation results"
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Failure	502	{object}	models.ErrorResponse
//	@Failure	503	{object}	models.ErrorResponse
//	@Router		/events/{id}/simulate-slots [post]
func SimulateSlots(c fiber.Ctx) error {
	return forwardToSimulator(c, "/simulate-slots", true)
}
