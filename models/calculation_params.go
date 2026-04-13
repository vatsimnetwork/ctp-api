package models

type SlotGenerationMode int

const (
	SlotGenerationModeRandom              SlotGenerationMode = iota
	SlotGenerationModeMaximizeAirportPairs
	SlotGenerationModeMaximizeSlots
)

type DepartureTimeWindowOffsetsCalculationMode int

const (
	DepartureTimeWindowOffsetsCalculationModeNone                    DepartureTimeWindowOffsetsCalculationMode = iota // 0
	DepartureTimeWindowOffsetsCalculationModeCalculateSlotTimingsOnly                                                // 1
	DepartureTimeWindowOffsetsCalculationModeEarliestRoutes                                                          // 2
	DepartureTimeWindowOffsetsCalculationModeLatestRoutes                                                            // 3
	DepartureTimeWindowOffsetsCalculationModeRouteAverage                                                            // 4
)

type WaypointThroughputCalculationMode int

const (
	WaypointThroughputCalculationModeNone WaypointThroughputCalculationMode = iota
	WaypointThroughputCalculationModeFirstWaypointsOfNATRouteSegmentsOnly
	WaypointThroughputCalculationModeAllWaypoints
)
