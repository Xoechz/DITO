using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using ValidationTracer.ApiService.Models;
using ValidationTracer.Data;

namespace ValidationTracer.ApiService.Controllers;

[ApiController]
[Route("[controller]")]
public class CostCenterController(ILogger<UserController> logger, ValidationTracerContext context) : ControllerBase
{
    private readonly ILogger<UserController> _logger = logger;
    private readonly ValidationTracerContext _context = context;

    [HttpGet]
    public async Task<IEnumerable<CostCenter>> Get()
    {
        _logger.LogInformation("Getting cost centers");
        return await _context.CostCenters
            .Select(c => new CostCenter { Code = c.Code })
            .ToListAsync();
    }
}