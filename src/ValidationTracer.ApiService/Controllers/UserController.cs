using Microsoft.AspNetCore.Mvc;
using ValidationTracer.Data.Models;
using ValidationTracer.Data.Repositories;

namespace ValidationTracer.ApiService.Controllers;

[ApiController]
[Route("[controller]")]
public class UserController(ILogger<UserController> logger, UserRepository userRepository)
    : ControllerBase
{
    #region Private Fields

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
        return await _userRepository.GetUsersAsync();
    }

    [HttpPost]
    public async Task Post(User user)
    {
        _logger.LogInformation("Adding user");
        await _userRepository.AddUserAsync(user);
    }

    [HttpPut]
    public async Task Put(User user)
    {
        _logger.LogInformation("Updating user");
        await _userRepository.UpdateUserAsync(user);
    }

    #endregion Public Methods
}