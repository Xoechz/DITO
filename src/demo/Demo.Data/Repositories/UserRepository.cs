using Demo.Data.Models;
using Demo.Data.Utilities;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;
using System.Diagnostics;

namespace Demo.Data.Repositories;

public class UserRepository(DemoContext demoContext, ILogger<UserRepository> logger, ActivitySource activitySource)
{
    #region Private Fields

    private readonly DemoContext _context = demoContext;
    private readonly ILogger<UserRepository> _logger = logger;
    private readonly ActivitySource _activitySource = activitySource;

    #endregion Private Fields

    #region Public Methods

    public async Task<bool> AddUsersAsync(IEnumerable<User> users)
    {
        var existingEmails = await _context.Users
            .Select(u => u.EmailAddress)
            .ToHashSetAsync();

        foreach (var user in users)
        {
            if (user.Error == ErrorType.Validation)
            {
                _logger.LogError("Validation error for user with email {EmailAddress}", user.EmailAddress);
                return false;
            }
            else if (user.Error == ErrorType.Critical)
            {
                throw new InvalidOperationException("Critical error occurred while adding user");
            }

            if (string.IsNullOrWhiteSpace(user.EmailAddress))
            {
                _logger.LogError("Email address is required");
                throw new InvalidOperationException("Email address cannot be null or empty");
            }

            if (existingEmails.Contains(user.EmailAddress))
            {
                _logger.LogWarning("User with email {EmailAddress} already exists", user.EmailAddress);
                return false;
            }

            var entity = new Entities.User
            {
                EmailAddress = user.EmailAddress
            };

            _context.Users.Add(entity);
        }

        await _context.SaveChangesAsync();
        return true;
    }

    public async Task DeleteUserAsync(string emailAddress)
    {
        var entity = await _context.Users.FirstOrDefaultAsync(user => user.EmailAddress == emailAddress);

        if (entity is not null)
        {
            _context.Users.Remove(entity);
            await _context.SaveChangesAsync();
        }
        else
        {
            throw new KeyNotFoundException($"User with email address {emailAddress} not found");
        }
    }

    public async Task<IEnumerable<User>> GetUsersAsync(IDictionary<ErrorType, decimal>? errorChances = null)
    {
        using var activity = _activitySource.StartActivity("GetUsers");

        var list = await _context.Users.Select(entity => new User
        {
            EmailAddress = entity.EmailAddress,
            Error = errorChances != null ? Utlis.GetRandomErrorType(errorChances) : ErrorType.None
        }).ToListAsync();

        activity?.SetTag("user.count", list.Count);
        return list;
    }

    #endregion Public Methods
}