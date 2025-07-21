using Microsoft.AspNetCore.Mvc;
using ValidationTracer.Data.Models;
using ValidationTracer.Data.Repositories;

namespace ValidationTracer.ApiService.Controllers;

[ApiController]
[Route("[controller]")]
public class CostCenterController(ILogger<UserController> logger, CostCenterRepository costCenterRepository) : ControllerBase
{
    #region Private Fields

    private readonly CostCenterRepository _costCenterRepository = costCenterRepository;
    private readonly ILogger<UserController> _logger = logger;

    #endregion Private Fields

    #region Public Methods

    [HttpGet]
    public async Task<IEnumerable<CostCenter>> Get()
    {
        _logger.LogInformation("Getting cost centers");
        return await _costCenterRepository.GetCostCentersAsync();
    }

    #endregion Public Methods
}