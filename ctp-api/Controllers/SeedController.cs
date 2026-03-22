using CTPSimulator;
using CTPSimulator.Context;
using Microsoft.AspNetCore.Mvc;

namespace ctp_api.Controllers
{
    [Route("api/[controller]")]
    [ApiController]
    public class SeedController : ControllerBase
    {
        private readonly AppDbContext _context;
        private readonly IWebHostEnvironment _env;

        public SeedController(AppDbContext context, IWebHostEnvironment env)
        {
            _context = context;
            _env = env;
        }

        [HttpPost]
        public async Task<IActionResult> Seed()
        {
            // Clear existing data
            _context.Slots.RemoveRange(_context.Slots);
            _context.RouteSegments.RemoveRange(_context.RouteSegments);
            _context.Airports.RemoveRange(_context.Airports);
            _context.Locations.RemoveRange(_context.Locations);
            _context.Sectors.RemoveRange(_context.Sectors);
            _context.VATSIMEvents.RemoveRange(_context.VATSIMEvents);
            await _context.SaveChangesAsync();

            // Load data from CSVs
            // Note: We need to make sure the paths are correct. 
            // The CSVs are in ../CTPSimulatorTestingApp/TestingData/ relative to the API project root?
            // Or we can just point to the absolute path for now if we know it.
            
            string projectRoot = Path.Combine(_env.ContentRootPath, "..");
            string testingDataPath = Path.Combine(projectRoot, "CTPSimulatorTestingApp", "TestingData");
            
            // Temporary change working directory or pass path to loader
            // Let's modify TestingDataLoader to accept a base path.
            
            var vatsimEvent = LoadWithCustomPath(testingDataPath, "25W");
            
            _context.VATSIMEvents.Add(vatsimEvent);
            await _context.SaveChangesAsync();

            return Ok(new { message = "Seeded successfully", eventId = vatsimEvent.Id });
        }

        private VATSIMEvent LoadWithCustomPath(string basePath, string eventPrefix)
        {
            VATSIMEvent vatsimEvent = new() 
            {
                Title = eventPrefix,
                Date = DateOnly.FromDateTime(DateTime.Now),
                DepartureTimeWindow = TimeSpan.FromHours(3)
            };


            Dictionary<string, Location> locations = new();
            Dictionary<string, Airport> airports = new();

            // read airports
            foreach (var line in System.IO.File.ReadAllLines(Path.Combine(basePath, $"{eventPrefix} Airports.csv")))
            {
                var splits = line.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
                if (splits.Length != 3) continue;
                var airport = new Airport
                {
                    Identifier = splits[0],
                    NumberOfVotes = ushort.Parse(splits[2]),
                    MaximumAircraftPerHour = (ushort)Math.Round(double.Parse(splits[1]) / 3),
                };

                airports.Add(airport.Identifier, airport);
                locations.Add(airport.Identifier, airport);

                vatsimEvent.Airports.Add(airport);
            }

            // read route segments
            foreach (var line in System.IO.File.ReadAllLines(Path.Combine(basePath, $"{eventPrefix} RouteSegments.csv")))
            {
                var splits = line.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
                if (splits.Length != 3) continue;
                var routeSegment = new RouteSegment() { Identifier = splits[0], RouteString = splits[1], RouteSegmentGroup = splits[2] };
                foreach (var waypoint in splits[1].Split(' ', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries))
                {
                    if (char.IsDigit(waypoint.Last()) || waypoint == "DCT") continue; // exclude airways and directs
                    if (!locations.TryGetValue(waypoint, out var location))
                    {
                        location = new Location() { Identifier = waypoint };
                        locations.Add(waypoint, location);
                        vatsimEvent.Waypoints.Add(location);
                    }
                    routeSegment.Locations.Add(location);
                }
                vatsimEvent.RouteSegments.Add(routeSegment);
            }

            return vatsimEvent;
        }
    }
}
