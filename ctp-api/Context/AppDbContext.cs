using Microsoft.EntityFrameworkCore;

namespace CTPSimulator.Context;

public class AppDbContext : DbContext
{
    public AppDbContext(DbContextOptions<AppDbContext> options)
        : base(options)
    {
    }

    public DbSet<VATSIMEvent> VATSIMEvents => Set<VATSIMEvent>();
    public DbSet<ThroughputPoint> ThroughputPoints => Set<ThroughputPoint>();
    public DbSet<Airport> Airports => Set<Airport>();
    public DbSet<Location> Locations => Set<Location>();
    public DbSet<RouteSegment> RouteSegments => Set<RouteSegment>();
    public DbSet<Sector> Sectors => Set<Sector>();
    public DbSet<Slot> Slots => Set<Slot>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.HasPostgresExtension("pgcrypto");

        // ---------- TPT inheritance for ThroughputPoint ----------
        // Base table holds shared columns; each subtype gets its own table.
        modelBuilder.Entity<ThroughputPoint>().UseTptMappingStrategy();
        modelBuilder.Entity<ThroughputPoint>().ToTable("ThroughputPoints");
        modelBuilder.Entity<Location>().ToTable("Locations");
        modelBuilder.Entity<Airport>().ToTable("Airports");
        modelBuilder.Entity<RouteSegment>().ToTable("RouteSegments");
        modelBuilder.Entity<Sector>().ToTable("Sectors");

        // ---------- VATSIMEvent ----------
        modelBuilder.Entity<VATSIMEvent>(entity =>
        {
            entity.HasMany(e => e.Airports).WithOne().HasForeignKey("VATSIMEventId");
            entity.HasMany(e => e.Waypoints).WithOne().HasForeignKey("VATSIMEventId");
            entity.HasMany(e => e.RouteSegments).WithOne().HasForeignKey("VATSIMEventId");
            entity.HasMany(e => e.Sectors).WithOne().HasForeignKey("VATSIMEventId");
            entity.HasMany(e => e.Slots).WithOne().HasForeignKey("VATSIMEventId");

            // SimulatorCalculationParameters stored as an owned type (same table, prefixed columns)
            entity.OwnsOne(e => e.CalculationParameters, cp =>
            {
                cp.Property(p => p.IntendedSlotGenerationMode).HasConversion<string>();
            });

            entity.Ignore(e => e.DepartureAirports);
            entity.Ignore(e => e.ArrivalAirports);
        });

        // ---------- ThroughputPoint (base) ----------
        modelBuilder.Entity<ThroughputPoint>(entity =>
        {
            entity.Ignore(e => e.SlotsStillAvailable);
            entity.Ignore(e => e.AreSlotsStillAvailable);
        });

        // ---------- Airport ----------
        modelBuilder.Entity<Airport>(entity =>
        {
            entity.Ignore(e => e.ConnectingPrimaryRouteSegments);
            entity.Ignore(e => e.ConnectingSecondaryRouteSegments);
        });

        // ---------- RouteSegment ----------
        modelBuilder.Entity<RouteSegment>(entity =>
        {
            // List<string> maps natively to PostgreSQL text[] via Npgsql
            entity.Property(e => e.RouteSegmentTags).HasColumnType("text[]");

            // Many-to-many: RouteSegment <-> Location (ordered waypoints along the route)
            entity.HasMany(e => e.Locations).WithMany()
                .UsingEntity("RouteSegmentLocations");

            // Many-to-many: RouteSegment <-> Sector (facility progression)
            entity.HasMany(e => e.ProvidedFacilityProgression).WithMany()
                .UsingEntity("RouteSegmentSectors");
        });

        // ---------- Sector ----------
        modelBuilder.Entity<Sector>(entity =>
        {
            // double[,] can't be stored directly — store as JSON instead
            entity.Property(e => e.Coordinates)
                .HasColumnType("jsonb")
                .HasConversion(
                    v => ConvertCoordinatesToJson(v),
                    v => ConvertJsonToCoordinates(v));
        });

        // ---------- Slot ----------
        modelBuilder.Entity<Slot>(entity =>
        {
            entity.HasOne(s => s.DepartureAirport).WithMany().HasForeignKey("DepartureAirportId");
            entity.HasOne(s => s.ArrivalAirport).WithMany().HasForeignKey("ArrivalAirportId");

            // Many-to-many: Slot <-> RouteSegment
            entity.HasMany(s => s.RouteSegments).WithMany()
                .UsingEntity("SlotRouteSegments");
        });
    }

    // ---------- Helpers for Sector.Coordinates (double[,] <-> JSON) ----------
    private static string ConvertCoordinatesToJson(double[,] coordinates)
    {
        if (coordinates == null) return "[]";
        int rows = coordinates.GetLength(0);
        int cols = coordinates.GetLength(1);
        var jagged = new double[rows][];
        for (int i = 0; i < rows; i++)
        {
            jagged[i] = new double[cols];
            for (int j = 0; j < cols; j++)
                jagged[i][j] = coordinates[i, j];
        }
        return System.Text.Json.JsonSerializer.Serialize(jagged);
    }

    private static double[,] ConvertJsonToCoordinates(string json)
    {
        if (string.IsNullOrEmpty(json)) return new double[0, 0];
        var jagged = System.Text.Json.JsonSerializer.Deserialize<double[][]>(json);
        if (jagged == null || jagged.Length == 0) return new double[0, 0];
        int rows = jagged.Length;
        int cols = jagged[0].Length;
        var result = new double[rows, cols];
        for (int i = 0; i < rows; i++)
            for (int j = 0; j < cols; j++)
                result[i, j] = jagged[i][j];
        return result;
    }
}