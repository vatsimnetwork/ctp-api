package handlers

import (
	"strings"
	"time"

	"github.com/vatsimnetwork/ctp-api/models"
)

// --- Airport ---

type airportInput struct {
	WaypointID             int64   `json:"waypointId"`
	Identifier             string  `json:"identifier"`
	Latitude               float64 `json:"latitude"`
	Longitude              float64 `json:"longitude"`
	MaximumAircraftPerHour uint16  `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16  `json:"maximumSlots"`
	NumberOfVotes          uint16  `json:"numberOfVotes"`
}

// --- Highlighted Waypoint ---

type highlightedWaypointInput struct {
	Identifier string  `json:"identifier"`
	Color      string  `json:"color"`
	Note       string  `json:"note"`
	WaypointID *int64  `json:"waypointId"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

// --- Event Tag Limit ---

type tagLimitResponse struct {
	Tag                    string `json:"tag"`
	MaximumAircraftPerHour uint16 `json:"maximumAircraftPerHour"`
}

// --- Route Segment ---

type locationInput struct {
	Identifier string  `json:"identifier"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	WaypointID int64   `json:"waypointId"`
	SortOrder  uint    `json:"sortOrder"`
}

type tagInput struct {
	Tag string `json:"tag"`
}

type routeSegmentInput struct {
	ID                          uint            `json:"id"`
	Identifier                  string          `json:"identifier"`
	MaximumAircraftPerHour      *uint16         `json:"maximumAircraftPerHour,omitempty"`
	RouteString                 string          `json:"routeString"`
	RouteSegmentGroup           string          `json:"routeSegmentGroup"`
	Color                       string          `json:"color"`
	Enabled                     bool            `json:"enabled"`
	Facilities                  string          `json:"facilities"`
	Tags                        []tagInput      `json:"tags"`
	ProvidedFacilityProgression []models.Sector `json:"providedFacilityProgression"`
	Locations                   []locationInput `json:"locations"`
	RouteRevision               uint            `json:"routeRevision"`
	EventID                     *uint           `json:"eventId,omitempty"`
}

// --- Simulator request structs ---

type simCalculationParameters struct {
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
	DepartureTimeWindowLength                             string  `json:"departureTimeWindowLength"`
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
	Id                       int64   `json:"id"`
	Identifier               string  `json:"identifier"`
	MaximumAircraftPerHour   uint16  `json:"maximumAircraftPerHour"`
	MaximumSlots             uint16  `json:"maximumSlots"`
	Latitude                 float64 `json:"latitude"`
	Longitude                float64 `json:"longitude"`
	NumberOfVotes            uint16  `json:"numberOfVotes"`
	DepartureTimeWindowStart *string `json:"departureTimeWindowStart,omitempty"`
}

type simSector struct {
	Id                     uint   `json:"id"`
	Identifier             string `json:"identifier"`
	MaximumAircraftPerHour uint16 `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16 `json:"maximumSlots"`
}

type simRouteSegment struct {
	Id                          uint    `json:"id"`
	Identifier                  string  `json:"identifier"`
	MaximumAircraftPerHour      uint16  `json:"maximumAircraftPerHour"`
	MaximumSlots                uint16  `json:"maximumSlots"`
	RouteString                 string  `json:"routeString"`
	RouteSegmentGroup           string  `json:"group"`
	Color                       string  `json:"color"`
	Enabled                     bool    `json:"enabled"`
	RouteSegmentTagIds          []uint  `json:"tagLimitIds"`
	ProvidedFacilityProgression []uint  `json:"providedFacilityProgressionIds"`
	Locations                   []int64 `json:"locationIds"`
	RouteRevision               uint    `json:"routeRevision"`
}

type simSlot struct {
	Id                   uint   `json:"id"`
	DepartureTime        string `json:"departureTime"`
	ProjectedArrivalTime string `json:"projectedArrivalTime"`
	DepartureAirport     int64  `json:"departureAirportId"`
	ArrivalAirport       int64  `json:"arrivalAirportId"`
	RouteSegments        []uint `json:"routeSegmentIds"`
}

type simTagLimit struct {
	Id           uint   `json:"id"`
	Tag          string `json:"identifier"`
	MaximumSlots uint16 `json:"maximumSlots"`
}

type simEvent struct {
	Id                                         uint                           `json:"id"`
	Title                                      string                         `json:"title"`
	RouteRevision                              uint                           `json:"routeRevision"`
	SlotRevision                               uint                           `json:"slotRevision"`
	Date                                       string                         `json:"date"`
	CalculationParameters                      simCalculationParameters       `json:"calculationParameters"`
	Airports                                   []simAirport                   `json:"airports"`
	Waypoints                                  []simWaypoint                  `json:"waypoints"`
	RouteSegments                              []simRouteSegment              `json:"routeSegments"`
	Sectors                                    []simSector                    `json:"sectors"`
	TagLimits                                  []simTagLimit                  `json:"tagLimits"`
	Slots                                      []simSlot                      `json:"slots"`
	AirportPairDepartureTimeWindowShiftingsIds map[int64]map[int64][2]float64 `json:"airportPairDepartureTimeWindowShiftingsIds"`
}

type simTime struct {
	time.Time
}

func (t *simTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.9999999Z07:00",
	} {
		if parsed, err := time.Parse(layout, s); err == nil {
			if parsed.Year() <= 1 {
				t.Time = time.Time{}
			} else {
				t.Time = parsed.UTC()
			}
			return nil
		}
	}
	t.Time = time.Time{}
	return nil
}

// --- Simulator response structs ---

type simResponseThroughput struct {
	Id             int64          `json:"id"`
	MaximumSlots   uint16         `json:"maximumSlots"`
	SlotsAllocated uint16         `json:"slotsAllocated"`
	SlotsFrames    map[int][]uint `json:"analysisFramesViaMinutesFromSynchronizationTimeSlotIds"`
}

type simResponseAirport struct {
	simResponseThroughput
	DepartureTimeWindowStart simTime `json:"departureTimeWindowStart"`
}

type simResponseSlot struct {
	Id                   uint                 `json:"id"`
	DepartureTime        simTime              `json:"departureTime"`
	ProjectedArrivalTime simTime              `json:"projectedArrivalTime"`
	DepartureAirport     int64                `json:"departureAirportId"`
	ArrivalAirport       int64                `json:"arrivalAirportId"`
	RouteSegments        []uint               `json:"routeSegmentIds"`
	SimulatedPositions   map[string][]float64 `json:"simulatedPositions,omitempty"`
}

type simResponseCalcParams struct {
	SlotGenerationOutputComments []string `json:"slotGenerationOutputComments"`
	SimulationOutputComments     []string `json:"simulationOutputComments"`
}

type simResponseEvent struct {
	CalculationParameters simResponseCalcParams   `json:"calculationParameters"`
	Airports              []simResponseAirport    `json:"airports"`
	Waypoints             []simResponseThroughput `json:"waypoints"`
	RouteSegments         []simResponseThroughput `json:"routeSegments"`
	Sectors               []simResponseThroughput `json:"sectors"`
	Slots                 []simResponseSlot       `json:"slots"`
}

// --- Throughput COPY helper ---

type snapshotRow struct {
	revisionID uint
	pointType  string
	pointID    int64
	minute     int
	slotID     uint
}

type positionRow struct {
	slotID    uint
	timestamp time.Time
	latitude  float64
	longitude float64
}

type slotPositionAtTime struct {
	SlotID             uint    `json:"slotId"`
	DepartureTime      string  `json:"departureTime"`
	ArrivalTime        string  `json:"arrivalTime"`
	DepartureAirport   string  `json:"departureAirport"`
	DepartureAirportID int64   `json:"departureAirportId"`
	ArrivalAirport     string  `json:"arrivalAirport"`
	ArrivalAirportID   int64   `json:"arrivalAirportId"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
}

type allSlotPositions struct {
	SlotID             uint                    `json:"slotId"`
	DepartureTime      string                  `json:"departureTime"`
	ArrivalTime        string                  `json:"arrivalTime"`
	DepartureAirport   string                  `json:"departureAirport"`
	DepartureAirportID int64                   `json:"departureAirportId"`
	ArrivalAirport     string                  `json:"arrivalAirport"`
	ArrivalAirportID   int64                   `json:"arrivalAirportId"`
	Positions          []slotPositionTimestamp `json:"positions"`
}

type slotPositionTimestamp struct {
	Timestamp string  `json:"timestamp"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// --- Slot Draft Entry ---

type SlotDraftEntryInput struct {
	DepartureAirportID uint `json:"departureAirportId"`
	DepRouteID         uint `json:"depRouteId"`
	TrackID            uint `json:"trackId"`
	ArrRouteID         uint `json:"arrRouteId"`
	ArrivalAirportID   uint `json:"arrivalAirportId"`
	SlotCount          uint `json:"slotCount"`
}

// --- Airport Pair Departure Window Shifts ---

type WindowShiftInput struct {
	DepartureAirportID uint    `json:"departureAirportId"`
	ArrivalAirportID   uint    `json:"arrivalAirportId"`
	StartShiftHours    float64 `json:"startShiftHours"`
	EndShiftHours      float64 `json:"endShiftHours"`
}

// --- Booking Import ---

type bookingRow struct {
	idx            int
	id             string
	vatsimID       string
	departure      string
	arrival        string
	oceanicTrack   string
	route          string
	takeOffTime    string
	flightLevel    string
	domesticFlight string
	selcalCode     string
}

type cityPair struct {
	dep string
	arr string
}

type bookingSlotInfo struct {
	slot     models.Slot
	orderMap map[uint]uint
	used     bool
}

type csvSlotRef struct {
	rowIdx int
	row    *bookingRow
}

type bookingMatchResult struct {
	bookingID uint
	slotID    uint
	track     string
	route     string
}
