using External.ApiService.Faker;
using External.ApiService.Models;
using Microsoft.AspNetCore.Mvc;

namespace External.ApiService.Controllers;

[ApiController]
[Route("[controller]")]
public class UserController(ILogger<UserController> logger,
                            ExternalApiUserFaker externalApiUserFaker)
    : ControllerBase
{
    private readonly ILogger<UserController> _logger = logger;
    private readonly List<ExternalApiUser> _users = externalApiUserFaker.Generate(1000);

    [HttpGet]
    public IEnumerable<ExternalApiUser> Get()
    {
        _logger.LogInformation("Getting users from external API");
        return _users;
    }
}