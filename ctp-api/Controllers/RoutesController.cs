using ctp_api.Context;
using CTPSimulator;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ctp_api.Controllers
{
    [Route("api/[controller]")]
    [ApiController]
    public class RoutesController : ControllerBase
    {
        private readonly AppDbContext _context;

        public RoutesController(AppDbContext context)
        {
            _context = context;
        }

        /// <summary>
        /// Get all route segments with their waypoints
        /// </summary>
        [HttpGet]
        public async Task<ActionResult<IEnumerable<RouteSegment>>> GetRoutes()
        {
            try
            {
                var routes = await _context.RouteSegments
                    .Include(r => r.Locations)
                    .OrderBy(r => r.RouteSegmentGroup)
                    .ThenBy(r => r.Identifier)
                    .ToListAsync();

                return Ok(routes);
            }
            catch (Exception ex)
            {
                return StatusCode(500, new { error = "Failed to retrieve routes", details = ex.Message });
            }
        }

        /// <summary>
        /// Get a specific route segment by identifier with all its waypoints
        /// </summary>
        [HttpGet("{identifier}")]
        public async Task<ActionResult<RouteSegment>> GetRoute(string identifier)
        {
            try
            {
                var route = await _context.RouteSegments
                    .Include(r => r.Locations)
                    .FirstOrDefaultAsync(r => r.Identifier == identifier);

                if (route == null)
                    return NotFound(new { error = $"Route '{identifier}' not found" });

                return Ok(route);
            }
            catch (Exception ex)
            {
                return StatusCode(500, new { error = "Failed to retrieve route", details = ex.Message });
            }
        }

        /// <summary>
        /// Create or update a route segment with waypoints
        /// Complete route data (RouteSegment + Waypoints) must be provided
        /// </summary>
        [HttpPost]
        public async Task<ActionResult> SaveRoute([FromBody] RouteSegment routeData)
        {
            if (routeData == null)
                return BadRequest(new { error = "Route data cannot be null" });

            if (string.IsNullOrWhiteSpace(routeData.Identifier))
                return BadRequest(new { error = "Route identifier is required" });

            if (string.IsNullOrWhiteSpace(routeData.RouteString))
                return BadRequest(new { error = "Route string is required" });

            if (routeData.Locations == null || routeData.Locations.Count == 0)
                return BadRequest(new { error = "Route must have at least one waypoint" });

            try
            {
                var existingRoute = await _context.RouteSegments
                    .Include(r => r.Locations)
                    .FirstOrDefaultAsync(r => r.Identifier == routeData.Identifier);

                if (existingRoute != null)
                {
                    // Update existing route
                    existingRoute.RouteString = routeData.RouteString;
                    existingRoute.RouteSegmentGroup = routeData.RouteSegmentGroup;
                    existingRoute.Color = routeData.Color;
                    existingRoute.Enabled = routeData.Enabled;
                    existingRoute.RouteSegmentTags = routeData.RouteSegmentTags;

                    // Update waypoints
                    existingRoute.Locations.Clear();
                    await _context.SaveChangesAsync();

                    foreach (var waypoint in routeData.Locations)
                    {
                        var location = await GetOrCreateLocation(waypoint);
                        existingRoute.Locations.Add(location);
                    }

                    _context.RouteSegments.Update(existingRoute);
                    await _context.SaveChangesAsync();

                    return Ok(new { message = "Route updated successfully", identifier = existingRoute.Identifier, id = existingRoute.Id });
                }
                else
                {
                    // Create new route
                    var newRoute = new RouteSegment
                    {
                        Identifier = routeData.Identifier,
                        RouteString = routeData.RouteString,
                        RouteSegmentGroup = routeData.RouteSegmentGroup,
                        Color = routeData.Color,
                        Enabled = routeData.Enabled,
                        RouteSegmentTags = routeData.RouteSegmentTags,
                        Locations = new List<Location>()
                    };

                    foreach (var waypoint in routeData.Locations)
                    {
                        var location = await GetOrCreateLocation(waypoint);
                        newRoute.Locations.Add(location);
                    }

                    _context.RouteSegments.Add(newRoute);
                    await _context.SaveChangesAsync();

                    return CreatedAtAction(nameof(GetRoute), new { identifier = newRoute.Identifier }, 
                        new { message = "Route created successfully", identifier = newRoute.Identifier, id = newRoute.Id });
                }
            }
            catch (Exception ex)
            {
                return StatusCode(500, new { error = "Failed to save route", details = ex.Message });
            }
        }

        /// <summary>
        /// Save multiple routes in a batch operation
        /// </summary>
        [HttpPost("batch")]
        public async Task<ActionResult> SaveRoutesBatch([FromBody] List<RouteSegment> routes)
        {
            if (routes == null || routes.Count == 0)
                return BadRequest(new { error = "Batch must contain at least one route" });

            var results = new List<object>();

            try
            {
                foreach (var routeData in routes)
                {
                    var existingRoute = await _context.RouteSegments
                        .Include(r => r.Locations)
                        .FirstOrDefaultAsync(r => r.Identifier == routeData.Identifier);

                    if (existingRoute != null)
                    {
                        existingRoute.RouteString = routeData.RouteString;
                        existingRoute.RouteSegmentGroup = routeData.RouteSegmentGroup;
                        existingRoute.Color = routeData.Color;
                        existingRoute.Enabled = routeData.Enabled;
                        existingRoute.RouteSegmentTags = routeData.RouteSegmentTags;

                        existingRoute.Locations.Clear();
                        await _context.SaveChangesAsync();

                        foreach (var waypoint in routeData.Locations)
                        {
                            var location = await GetOrCreateLocation(waypoint);
                            existingRoute.Locations.Add(location);
                        }

                        _context.RouteSegments.Update(existingRoute);
                    }
                    else
                    {
                        var newRoute = new RouteSegment
                        {
                            Identifier = routeData.Identifier,
                            RouteString = routeData.RouteString,
                            RouteSegmentGroup = routeData.RouteSegmentGroup,
                            Color = routeData.Color,
                            Enabled = routeData.Enabled,
                            RouteSegmentTags = routeData.RouteSegmentTags,
                            Locations = new List<Location>()
                        };

                        foreach (var waypoint in routeData.Locations)
                        {
                            var location = await GetOrCreateLocation(waypoint);
                            newRoute.Locations.Add(location);
                        }

                        _context.RouteSegments.Add(newRoute);
                    }

                    results.Add(new { identifier = routeData.Identifier, status = "processed" });
                }

                await _context.SaveChangesAsync();
                return Ok(new { message = $"Batch saved successfully: {results.Count} routes processed", results });
            }
            catch (Exception ex)
            {
                return StatusCode(500, new { error = "Failed to save batch routes", details = ex.Message });
            }
        }

        /// <summary>
        /// Delete a route segment by identifier
        /// </summary>
        [HttpDelete("{identifier}")]
        public async Task<ActionResult> DeleteRoute(string identifier)
        {
            try
            {
                var route = await _context.RouteSegments.FirstOrDefaultAsync(r => r.Identifier == identifier);

                if (route == null)
                    return NotFound(new { error = $"Route '{identifier}' not found" });

                _context.RouteSegments.Remove(route);
                await _context.SaveChangesAsync();

                return Ok(new { message = $"Route '{identifier}' deleted successfully" });
            }
            catch (Exception ex)
            {
                return StatusCode(500, new { error = "Failed to delete route", details = ex.Message });
            }
        }

        #region Helper Methods

        /// <summary>
        /// Gets an existing Location or creates a new one
        /// </summary>
        private async Task<Location> GetOrCreateLocation(Location waypointData)
        {
            var existing = await _context.Locations
                .FirstOrDefaultAsync(l => l.Identifier == waypointData.Identifier);

            if (existing != null)
            {
                // Update if values differ
                if (existing.Latitude != waypointData.Latitude || 
                    existing.Longitude != waypointData.Longitude ||
                    existing.MaximumAircraftPerHour != waypointData.MaximumAircraftPerHour)
                {
                    existing.Latitude = waypointData.Latitude;
                    existing.Longitude = waypointData.Longitude;
                    existing.MaximumAircraftPerHour = waypointData.MaximumAircraftPerHour;
                    _context.Locations.Update(existing);
                }
                return existing;
            }

            var newLocation = new Location
            {
                Identifier = waypointData.Identifier,
                Latitude = waypointData.Latitude,
                Longitude = waypointData.Longitude,
                MaximumAircraftPerHour = waypointData.MaximumAircraftPerHour
            };

            _context.Locations.Add(newLocation);
            await _context.SaveChangesAsync();
            return newLocation;
        }

        #endregion
    }
}
