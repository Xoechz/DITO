using System.Diagnostics;
using Demo.Data.Models;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace Demo.Data.Repositories;

public class UserRepository(DemoContext demoContext, ILogger<UserRepository> logger, ActivitySource activitySource)
{
    #region Private Fields

    private readonly DemoContext _context = demoContext;
    private readonly ILogger<UserRepository> _logger = logger;
    private readonly ActivitySource _activitySource = activitySource;

    #endregion Private Fields

    #region Public Methods

    public async Task<bool> AddUserAsync(User user)
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

        var userExists = await _context.Users.AnyAsync(u => u.EmailAddress == user.EmailAddress);

        if (userExists)
        {
            _logger.LogWarning("User with email {EmailAddress} already exists", user.EmailAddress);
            return false;
        }

        var entity = new Entities.User
        {
            EmailAddress = user.EmailAddress
        };

        _context.Users.Add(entity);
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
            Error = errorChances != null ? GetRandomErrorType(errorChances) : ErrorType.None
        }).ToListAsync();

        activity?.SetTag("user.count", list.Count);
        return list;
    }

    #endregion Public Methods

    #region Private Methods

    private static ErrorType GetRandomErrorType(IDictionary<ErrorType, decimal> errorChances)
    {
        var randomValue = (decimal)new Random().NextDouble();
        var cumulativeChance = 0.0m;

        foreach (var kvp in errorChances)
        {
            cumulativeChance += kvp.Value;
            if (randomValue < cumulativeChance)
            {
                return kvp.Key;
            }
        }

        return ErrorType.None; // Default case if no error type matches
    }

    #endregion Private Methods
}