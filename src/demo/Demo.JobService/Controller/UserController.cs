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

    [HttpDelete]
    public async Task Delete(string emailAddress)
    {
        _logger.LogInformation("Deleting user");
        await _userRepository.DeleteUserAsync(emailAddress);
    }

    [HttpGet]
    public async Task<IEnumerable<User>> Get()
    {
        _logger.LogInformation("Getting users");
        return await _userRepository.GetUsersAsync(_config.ErrorChances);
    }

    [HttpPost]
    public async Task Post(User user)
    {
        _logger.LogInformation("Adding user");
        await _userRepository.AddUserAsync(user);
    }

    #endregion Public Methods
}