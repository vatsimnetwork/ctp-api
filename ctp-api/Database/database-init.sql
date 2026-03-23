-- Database initialisation script
-- Version 1.0

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE SCHEMA IF NOT EXISTS ctpapi;

CREATE TABLE IF NOT EXISTS ctpapi.ThroughputPoint (
    ID uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    MaximumAircraftPerHour INTEGER NOT NULL,
    SlotsAllocated INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS ctpapi.Event (
    EventID uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    Title TEXT NOT NULL,
    RouteRevision INTEGER NOT NULL,
    SlotRevision INTEGER NOT NULL,

    SynchronizationDateTime TIMESTAMP NOT NULL,
    SynchronizationLongitude DOUBLE PRECISION NOT NULL,

    DepartureTimeStart TIMESTAMP NOT NULL,
    DepartureTimeEnd TIMESTAMP NOT NULL,

    SimulationTimeResolutionStart TIMESTAMP NOT NULL,
    SimulationTimeResolutionEnd TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS ctpapi.Locations (
    LocationID uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    EventID uuid REFERENCES ctpapi.Event(EventID) NOT NULL,
    Identifier TEXT NOT NULL,
    Latitude DOUBLE PRECISION NOT NULL,
    Longitude DOUBLE PRECISION NOT NULL,
    MaxAircraftPerHour INTEGER NOT NULL,
    IsAirport BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS ctpapi.RouteSegments (
    RouteSegmentID UUID PRIMARY KEY  NOT NULL DEFAULT gen_random_uuid(),
    EventID uuid REFERENCES ctpapi.Event(EventID) NOT NULL,
    Identifier TEXT NOT NULL,
    RouteString TEXT NOT NULL,
    MaxAircraftPerHour INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS ctpapi.RouteSegmentLocations (
    RouteSegmentLocationID UUID PRIMARY KEY  NOT NULL DEFAULT gen_random_uuid(),
    RouteSegmentID UUID REFERENCES ctpapi.RouteSegments(RouteSegmentID) NOT NULL,
    LocationID UUID REFERENCES ctpapi.Locations(LocationID) NOT NULL,
    SequenceOrder INTEGER NOT NULL
);
