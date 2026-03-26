using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using CTPSimulator;
using ctp_api.Context;

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
        public async Task<ActionResult<VATSIMEvent>> GetVATSIMEvent(uint id)
        {
            var vATSIMEvent = await _context.VATSIMEvents.FindAsync(id);

            if (vATSIMEvent == null)
            {
                return NotFound();
            }

            return vATSIMEvent;
        }

        // PUT: api/VATSIMEvent/5
        // To protect from overposting attacks, see https://go.microsoft.com/fwlink/?linkid=2123754
        [HttpPut("{id}")]
        public async Task<IActionResult> PutVATSIMEvent(uint id, VATSIMEvent vATSIMEvent)
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
        public async Task<IActionResult> DeleteVATSIMEvent(uint id)
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

        private bool VATSIMEventExists(uint id)
        {
            return _context.VATSIMEvents.Any(e => e.Id == id);
        }
    }
}
