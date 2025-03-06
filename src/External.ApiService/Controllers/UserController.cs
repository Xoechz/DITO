using External.ApiService.Models;
using Microsoft.AspNetCore.Mvc;

namespace External.ApiService.Controllers;

[ApiController]
[Route("[controller]")]
public class UserController(ILogger<UserController> logger) : ControllerBase
{
    private readonly ILogger<UserController> _logger = logger;

    [HttpGet]
    public IEnumerable<ExternalApiUser> Get()
    {
        _logger.LogInformation("Getting users from external API");
        return Enumerable.Range(1, 5).Select(index => new ExternalApiUser
        {
            EmailAddress = $"{index}",
            ExternalApiProperty = "ExternalApiProperty",
            CostCenterCode = "CostCenterCode"
        });
    }
}