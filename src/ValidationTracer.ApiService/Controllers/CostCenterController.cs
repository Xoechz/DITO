using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using ValidationTracer.ApiService.Models;
using ValidationTracer.Data;

namespace ValidationTracer.ApiService.Controllers;

[ApiController]
[Route("[controller]")]
public class CostCenterController(ILogger<UserController> logger, ValidationTracerContext context) : ControllerBase
{
    #region Private Fields

    private readonly ValidationTracerContext _context = context;
    private readonly ILogger<UserController> _logger = logger;

    #endregion Private Fields

    #region Public Methods

    [HttpGet]
    public async Task<IEnumerable<CostCenter>> Get()
    {
        _logger.LogInformation("Getting cost centers");
        return await _context.CostCenters
            .Select(c => new CostCenter { Code = c.Code })
            .ToListAsync();
    }

    #endregion Public Methods
}