using Microsoft.EntityFrameworkCore.Migrations;

#nullable disable

namespace CTPSimulator.Migrations
{
    public partial class AddRouteSegmentEnabledAndColor : Migration
    {
        protected override void Up(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.AddColumn<string>(
                name: "Color",
                table: "RouteSegments",
                type: "character varying(20)",
                maxLength: 20,
                nullable: false,
                defaultValue: "");

            migrationBuilder.AddColumn<bool>(
                name: "Enabled",
                table: "RouteSegments",
                type: "boolean",
                nullable: false,
                defaultValue: true);

            migrationBuilder.CreateIndex(
                name: "IX_RouteSegments_Enabled",
                table: "RouteSegments",
                column: "Enabled");
        }

        protected override void Down(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.DropIndex(
                name: "IX_RouteSegments_Enabled",
                table: "RouteSegments");

            migrationBuilder.DropColumn(
                name: "Color",
                table: "RouteSegments");

            migrationBuilder.DropColumn(
                name: "Enabled",
                table: "RouteSegments");
        }
    }
}
