using CTPSimulator;
using CTPSimulator.Context;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ctp_api.Controllers
{
    [Route("api/[controller]")]
    [ApiController]
    public class VATSIMEventController : ControllerBase
    {
        private readonly AppDbContext _context;

        public VATSIMEventController(AppDbContext context)
        {
            _context = context;
        }

        // GET: api/VATSIMEvent
        [HttpGet]
        public async Task<ActionResult<IEnumerable<VATSIMEvent>>> GetVATSIMEvents()
        {
            return await _context.VATSIMEvents.ToListAsync();
        }

        // GET: api/VATSIMEvent/5
        [HttpGet("{id}")]
        public async Task<ActionResult<VATSIMEvent>> GetVATSIMEvent(int id)
        {
            var vATSIMEvent = await _context.VATSIMEvents
                .Include(e => e.Airports)
                .Include(e => e.Waypoints)
                .Include(e => e.RouteSegments)
                .Include(e => e.Sectors)
                .FirstOrDefaultAsync(e => e.Id == id);

            if (vATSIMEvent == null)
            {
                return NotFound();
            }

            return vATSIMEvent;
        }

        // PUT: api/VATSIMEvent/5
        // To protect from overposting attacks, see https://go.microsoft.com/fwlink/?linkid=2123754
        [HttpPut("{id}")]
        public async Task<IActionResult> PutVATSIMEvent(int id, VATSIMEvent vATSIMEvent)
        {
            if (id != vATSIMEvent.Id)
            {
                return BadRequest();
            }

            _context.Entry(vATSIMEvent).State = EntityState.Modified;

            try
            {
                await _context.SaveChangesAsync();
            }
            catch (DbUpdateConcurrencyException)
            {
                if (!VATSIMEventExists(id))
                {
                    return NotFound();
                }
                else
                {
                    throw;
                }
            }

            return NoContent();
        }

        // POST: api/VATSIMEvent
        // To protect from overposting attacks, see https://go.microsoft.com/fwlink/?linkid=2123754
        [HttpPost]
        public async Task<ActionResult<VATSIMEvent>> PostVATSIMEvent(VATSIMEvent vATSIMEvent)
        {
            _context.VATSIMEvents.Add(vATSIMEvent);
            await _context.SaveChangesAsync();

            return CreatedAtAction("GetVATSIMEvent", new { id = vATSIMEvent.Id }, vATSIMEvent);
        }

        // DELETE: api/VATSIMEvent/5
        [HttpDelete("{id}")]
        public async Task<IActionResult> DeleteVATSIMEvent(int id)
        {
            var vATSIMEvent = await _context.VATSIMEvents.FindAsync(id);
            if (vATSIMEvent == null)
            {
                return NotFound();
            }

            _context.VATSIMEvents.Remove(vATSIMEvent);
            await _context.SaveChangesAsync();

            return NoContent();
        }

        private bool VATSIMEventExists(int id)
        {
            return _context.VATSIMEvents.Any(e => e.Id == id);
        }
    }
}
