using ctp_api.Context;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ctp_api.Controllers
{
    [Route("api/[controller]")]
    [ApiController]
    public class ThroughputPointsController : ControllerBase
    {
        private readonly AppDbContext _context;

        public ThroughputPointsController(AppDbContext context)
        {
            _context = context;
        }

        // GET: api/ThroughputPoints
        [HttpGet]
        public async Task<ActionResult<IEnumerable<CTPSimulator.ThroughputPoint>>> GetThroughputPoints()
        {
            return await _context.ThroughputPoints.ToListAsync();
        }

        // GET: api/ThroughputPoints/5
        [HttpGet("{id}")]
        public async Task<ActionResult<CTPSimulator.ThroughputPoint>> GetThroughputPoint(int id)
        {
            var throughputPoint = await _context.ThroughputPoints.FindAsync(id);

            if (throughputPoint == null)
            {
                return NotFound();
            }

            return throughputPoint;
        }

        // PUT: api/ThroughputPoints/5
        // To protect from overposting attacks, see https://go.microsoft.com/fwlink/?linkid=2123754
        [HttpPut("{id}")]
        public async Task<IActionResult> PutThroughputPoint(int id, CTPSimulator.ThroughputPoint throughputPoint)
        {
            if (id != throughputPoint.Id)
            {
                return BadRequest();
            }

            _context.Entry(throughputPoint).State = EntityState.Modified;

            try
            {
                await _context.SaveChangesAsync();
            }
            catch (DbUpdateConcurrencyException)
            {
                if (!ThroughputPointExists(id))
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

        // POST: api/ThroughputPoints
        // To protect from overposting attacks, see https://go.microsoft.com/fwlink/?linkid=2123754
        [HttpPost]
        public async Task<ActionResult<CTPSimulator.ThroughputPoint>> PostThroughputPoint(CTPSimulator.ThroughputPoint throughputPoint)
        {
            _context.ThroughputPoints.Add(throughputPoint);
            await _context.SaveChangesAsync();

            return CreatedAtAction("GetThroughputPoint", new { id = throughputPoint.Id }, throughputPoint);
        }

        // DELETE: api/ThroughputPoints/5
        [HttpDelete("{id}")]
        public async Task<IActionResult> DeleteThroughputPoint(int id)
        {
            var throughputPoint = await _context.ThroughputPoints.FindAsync(id);
            if (throughputPoint == null)
            {
                return NotFound();
            }

            _context.ThroughputPoints.Remove(throughputPoint);
            await _context.SaveChangesAsync();

            return NoContent();
        }

        private bool ThroughputPointExists(int id)
        {
            return _context.ThroughputPoints.Any(e => e.Id == id);
        }
    }
}
