using Demo.Data.Models;
using Demo.Data.Repositories;
using Demo.JobService.Config;
using Microsoft.AspNetCore.Mvc;

namespace Demo.JobService.Controller;

[ApiController]
[Route("[controller]")]
public class UserController(ILogger<UserController> logger,
                            UserRepository userRepository,
                            JobConfig config)
    : ControllerBase
{
    #region Private Fields

    private readonly JobConfig _config = config;
    private readonly ILogger<UserController> _logger = logger;
    private readonly UserRepository _userRepository = userRepository;

    #endregion Private Fields

    #region Public Methods

    [HttpDelete("{emailAddress}")]
    public async Task<IActionResult> Delete(string emailAddress)
    {
        _logger.LogInformation("Deleting user");

        try
        {
            await _userRepository.DeleteUserAsync(emailAddress);
        }
        catch (KeyNotFoundException)
        {
            return NotFound();
        }

        return Ok();
    }

    [HttpGet]
    public async Task<IEnumerable<User>> Get()
    {
        _logger.LogInformation("Getting users");
        return await _userRepository.GetUsersAsync(_config.ErrorChances);
    }

    [HttpPost]
    public async Task Post(IEnumerable<User> users)
    {
        _logger.LogInformation("Adding users");
        await _userRepository.AddUsersAsync(users);
    }

    #endregion Public Methods
}