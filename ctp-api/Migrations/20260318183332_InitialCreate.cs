using System;
using System.Collections.Generic;
using Microsoft.EntityFrameworkCore.Migrations;
using Npgsql.EntityFrameworkCore.PostgreSQL.Metadata;

#nullable disable

namespace CTPSimulator.Migrations
{
    /// <inheritdoc />
    public partial class InitialCreate : Migration
    {
        /// <inheritdoc />
        protected override void Up(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.AlterDatabase()
                .Annotation("Npgsql:PostgresExtension:pgcrypto", ",,");

            migrationBuilder.CreateTable(
                name: "ThroughputPoints",
                columns: table => new
                {
                    Id = table.Column<int>(type: "integer", nullable: false)
                        .Annotation("Npgsql:ValueGenerationStrategy", NpgsqlValueGenerationStrategy.IdentityByDefaultColumn),
                    Identifier = table.Column<string>(type: "text", nullable: false),
                    MaximumAircraftPerHour = table.Column<int>(type: "integer", nullable: false),
                    MaximumSlots = table.Column<int>(type: "integer", nullable: false),
                    SlotsAllocated = table.Column<int>(type: "integer", nullable: false),
                    SimulationAnalysisStartTime = table.Column<DateTime>(type: "timestamp with time zone", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ThroughputPoints", x => x.Id);
                });

            migrationBuilder.CreateTable(
                name: "VATSIMEvents",
                columns: table => new
                {
                    Id = table.Column<int>(type: "integer", nullable: false)
                        .Annotation("Npgsql:ValueGenerationStrategy", NpgsqlValueGenerationStrategy.IdentityByDefaultColumn),
                    Title = table.Column<string>(type: "text", nullable: false),
                    RouteRevision = table.Column<long>(type: "bigint", nullable: false),
                    SlotRevision = table.Column<long>(type: "bigint", nullable: false),
                    Date = table.Column<DateOnly>(type: "date", nullable: false),
                    DepartureTimeWindow = table.Column<TimeSpan>(type: "interval", nullable: false),
                    CalculationParameters_RecalculateMaximumAirportSlots = table.Column<bool>(type: "boolean", nullable: false),
                    CalculationParameters_IntendedSlotGenerationMode = table.Column<string>(type: "text", nullable: false),
                    CalculationParameters_SlotGenerationOutputCommentary = table.Column<string>(type: "text", nullable: false),
                    CalculationParameters_DepartureTimeWindowOffsetSynchronization = table.Column<double>(name: "CalculationParameters_DepartureTimeWindowOffsetSynchronization~", type: "double precision", nullable: false),
                    CalculationParameters_SimulationAnalysisResolution = table.Column<TimeSpan>(type: "interval", nullable: false),
                    CalculationParameters_ShouldSimulationUseActualWeatherForecast = table.Column<bool>(name: "CalculationParameters_ShouldSimulationUseActualWeatherForecast~", type: "boolean", nullable: false),
                    CalculationParameters_SimulationOutputCommentary = table.Column<string>(type: "text", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_VATSIMEvents", x => x.Id);
                });

            migrationBuilder.CreateTable(
                name: "Locations",
                columns: table => new
                {
                    Id = table.Column<int>(type: "integer", nullable: false),
                    Latitude = table.Column<double>(type: "double precision", nullable: false),
                    Longitude = table.Column<double>(type: "double precision", nullable: false),
                    VATSIMEventId = table.Column<int>(type: "integer", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Locations", x => x.Id);
                    table.ForeignKey(
                        name: "FK_Locations_ThroughputPoints_Id",
                        column: x => x.Id,
                        principalTable: "ThroughputPoints",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_Locations_VATSIMEvents_VATSIMEventId",
                        column: x => x.VATSIMEventId,
                        principalTable: "VATSIMEvents",
                        principalColumn: "Id");
                });

            migrationBuilder.CreateTable(
                name: "RouteSegments",
                columns: table => new
                {
                    Id = table.Column<int>(type: "integer", nullable: false),
                    RouteString = table.Column<string>(type: "text", nullable: false),
                    RouteSegmentGroup = table.Column<string>(type: "text", nullable: false),
                    RouteSegmentTags = table.Column<List<string>>(type: "text[]", nullable: false),
                    VATSIMEventId = table.Column<int>(type: "integer", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_RouteSegments", x => x.Id);
                    table.ForeignKey(
                        name: "FK_RouteSegments_ThroughputPoints_Id",
                        column: x => x.Id,
                        principalTable: "ThroughputPoints",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_RouteSegments_VATSIMEvents_VATSIMEventId",
                        column: x => x.VATSIMEventId,
                        principalTable: "VATSIMEvents",
                        principalColumn: "Id");
                });

            migrationBuilder.CreateTable(
                name: "Sectors",
                columns: table => new
                {
                    Id = table.Column<int>(type: "integer", nullable: false),
                    MaxLatitude = table.Column<double>(type: "double precision", nullable: false),
                    MinLatitude = table.Column<double>(type: "double precision", nullable: false),
                    MaxLongitude = table.Column<double>(type: "double precision", nullable: false),
                    MinLongitude = table.Column<double>(type: "double precision", nullable: false),
                    Coordinates = table.Column<string>(type: "jsonb", nullable: false),
                    VATSIMEventId = table.Column<int>(type: "integer", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Sectors", x => x.Id);
                    table.ForeignKey(
                        name: "FK_Sectors_ThroughputPoints_Id",
                        column: x => x.Id,
                        principalTable: "ThroughputPoints",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_Sectors_VATSIMEvents_VATSIMEventId",
                        column: x => x.VATSIMEventId,
                        principalTable: "VATSIMEvents",
                        principalColumn: "Id");
                });

            migrationBuilder.CreateTable(
                name: "Airports",
                columns: table => new
                {
                    Id = table.Column<int>(type: "integer", nullable: false),
                    DepartureTimeWindowStart = table.Column<DateTime>(type: "timestamp with time zone", nullable: false),
                    NumberOfVotes = table.Column<int>(type: "integer", nullable: false),
                    VATSIMEventId1 = table.Column<int>(type: "integer", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Airports", x => x.Id);
                    table.ForeignKey(
                        name: "FK_Airports_Locations_Id",
                        column: x => x.Id,
                        principalTable: "Locations",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_Airports_VATSIMEvents_VATSIMEventId1",
                        column: x => x.VATSIMEventId1,
                        principalTable: "VATSIMEvents",
                        principalColumn: "Id");
                });

            migrationBuilder.CreateTable(
                name: "RouteSegmentLocations",
                columns: table => new
                {
                    LocationsId = table.Column<int>(type: "integer", nullable: false),
                    RouteSegmentId = table.Column<int>(type: "integer", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_RouteSegmentLocations", x => new { x.LocationsId, x.RouteSegmentId });
                    table.ForeignKey(
                        name: "FK_RouteSegmentLocations_Locations_LocationsId",
                        column: x => x.LocationsId,
                        principalTable: "Locations",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_RouteSegmentLocations_RouteSegments_RouteSegmentId",
                        column: x => x.RouteSegmentId,
                        principalTable: "RouteSegments",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "RouteSegmentSectors",
                columns: table => new
                {
                    ProvidedFacilityProgressionId = table.Column<int>(type: "integer", nullable: false),
                    RouteSegmentId = table.Column<int>(type: "integer", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_RouteSegmentSectors", x => new { x.ProvidedFacilityProgressionId, x.RouteSegmentId });
                    table.ForeignKey(
                        name: "FK_RouteSegmentSectors_RouteSegments_RouteSegmentId",
                        column: x => x.RouteSegmentId,
                        principalTable: "RouteSegments",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_RouteSegmentSectors_Sectors_ProvidedFacilityProgressionId",
                        column: x => x.ProvidedFacilityProgressionId,
                        principalTable: "Sectors",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "Slots",
                columns: table => new
                {
                    Id = table.Column<int>(type: "integer", nullable: false)
                        .Annotation("Npgsql:ValueGenerationStrategy", NpgsqlValueGenerationStrategy.IdentityByDefaultColumn),
                    DepartureTime = table.Column<DateTime>(type: "timestamp with time zone", nullable: false),
                    ProjectedArrivalTime = table.Column<DateTime>(type: "timestamp with time zone", nullable: false),
                    DepartureAirportId = table.Column<int>(type: "integer", nullable: false),
                    ArrivalAirportId = table.Column<int>(type: "integer", nullable: false),
                    ThroughputPointId = table.Column<int>(type: "integer", nullable: true),
                    VATSIMEventId = table.Column<int>(type: "integer", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Slots", x => x.Id);
                    table.ForeignKey(
                        name: "FK_Slots_Airports_ArrivalAirportId",
                        column: x => x.ArrivalAirportId,
                        principalTable: "Airports",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_Slots_Airports_DepartureAirportId",
                        column: x => x.DepartureAirportId,
                        principalTable: "Airports",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_Slots_ThroughputPoints_ThroughputPointId",
                        column: x => x.ThroughputPointId,
                        principalTable: "ThroughputPoints",
                        principalColumn: "Id");
                    table.ForeignKey(
                        name: "FK_Slots_VATSIMEvents_VATSIMEventId",
                        column: x => x.VATSIMEventId,
                        principalTable: "VATSIMEvents",
                        principalColumn: "Id");
                });

            migrationBuilder.CreateTable(
                name: "SlotRouteSegments",
                columns: table => new
                {
                    RouteSegmentsId = table.Column<int>(type: "integer", nullable: false),
                    SlotId = table.Column<int>(type: "integer", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_SlotRouteSegments", x => new { x.RouteSegmentsId, x.SlotId });
                    table.ForeignKey(
                        name: "FK_SlotRouteSegments_RouteSegments_RouteSegmentsId",
                        column: x => x.RouteSegmentsId,
                        principalTable: "RouteSegments",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_SlotRouteSegments_Slots_SlotId",
                        column: x => x.SlotId,
                        principalTable: "Slots",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateIndex(
                name: "IX_Airports_VATSIMEventId1",
                table: "Airports",
                column: "VATSIMEventId1");

            migrationBuilder.CreateIndex(
                name: "IX_Locations_VATSIMEventId",
                table: "Locations",
                column: "VATSIMEventId");

            migrationBuilder.CreateIndex(
                name: "IX_RouteSegmentLocations_RouteSegmentId",
                table: "RouteSegmentLocations",
                column: "RouteSegmentId");

            migrationBuilder.CreateIndex(
                name: "IX_RouteSegments_VATSIMEventId",
                table: "RouteSegments",
                column: "VATSIMEventId");

            migrationBuilder.CreateIndex(
                name: "IX_RouteSegmentSectors_RouteSegmentId",
                table: "RouteSegmentSectors",
                column: "RouteSegmentId");

            migrationBuilder.CreateIndex(
                name: "IX_Sectors_VATSIMEventId",
                table: "Sectors",
                column: "VATSIMEventId");

            migrationBuilder.CreateIndex(
                name: "IX_SlotRouteSegments_SlotId",
                table: "SlotRouteSegments",
                column: "SlotId");

            migrationBuilder.CreateIndex(
                name: "IX_Slots_ArrivalAirportId",
                table: "Slots",
                column: "ArrivalAirportId");

            migrationBuilder.CreateIndex(
                name: "IX_Slots_DepartureAirportId",
                table: "Slots",
                column: "DepartureAirportId");

            migrationBuilder.CreateIndex(
                name: "IX_Slots_ThroughputPointId",
                table: "Slots",
                column: "ThroughputPointId");

            migrationBuilder.CreateIndex(
                name: "IX_Slots_VATSIMEventId",
                table: "Slots",
                column: "VATSIMEventId");
        }

        /// <inheritdoc />
        protected override void Down(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.DropTable(
                name: "RouteSegmentLocations");

            migrationBuilder.DropTable(
                name: "RouteSegmentSectors");

            migrationBuilder.DropTable(
                name: "SlotRouteSegments");

            migrationBuilder.DropTable(
                name: "Sectors");

            migrationBuilder.DropTable(
                name: "RouteSegments");

            migrationBuilder.DropTable(
                name: "Slots");

            migrationBuilder.DropTable(
                name: "Airports");

            migrationBuilder.DropTable(
                name: "Locations");

            migrationBuilder.DropTable(
                name: "ThroughputPoints");

            migrationBuilder.DropTable(
                name: "VATSIMEvents");
        }
    }
}
