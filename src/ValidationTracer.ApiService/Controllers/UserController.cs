using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using ValidationTracer.ApiService.Models;
using ValidationTracer.Data;

namespace ValidationTracer.ApiService.Controllers;

[ApiController]
[Route("[controller]")]
public class UserController(ILogger<UserController> logger, ValidationTracerContext context) : ControllerBase
{
    #region Private Fields

    private readonly ValidationTracerContext _context = context;
    private readonly ILogger<UserController> _logger = logger;

    #endregion Private Fields

    #region Public Methods

    [HttpGet]
    public async Task<IEnumerable<User>> Get()
    {
        _logger.LogInformation("Getting users");
        return await _context.Users
            .Select(u => new User
            {
                EmailAddress = u.EmailAddress,
                ExternalDbProperty = u.ExternalDbProperty,
                ExternalApiProperty = u.ExternalApiProperty,
                InternalProperty = u.InternalProperty,
                CostCenterCode = u.CostCenterCode
            })
            .ToListAsync();
    }

    [HttpPost]
    public async Task<IActionResult> Post(User user)
    {
        _logger.LogInformation("Adding user");
        _context.Users.Add(new Data.Entities.User
        {
            EmailAddress = user.EmailAddress,
            ExternalDbProperty = user.ExternalDbProperty,
            ExternalApiProperty = user.ExternalApiProperty,
            InternalProperty = user.InternalProperty,
            CostCenterCode = user.CostCenterCode
        });
        await _context.SaveChangesAsync();
        return Ok();
    }

    [HttpPut]
    public async Task<IActionResult> Put(User user)
    {
        _logger.LogInformation("Updating user");
        var userToUpdate = await _context.Users.FindAsync(user.EmailAddress);

        if (userToUpdate == null)
        {
            return NotFound();
        }

        userToUpdate.ExternalDbProperty = user.ExternalDbProperty;
        userToUpdate.ExternalApiProperty = user.ExternalApiProperty;
        userToUpdate.InternalProperty = user.InternalProperty;
        userToUpdate.CostCenterCode = user.CostCenterCode;

        await _context.SaveChangesAsync();
        return Ok();
    }

    #endregion Public Methods
}