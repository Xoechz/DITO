using Demo.Data.Models;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace Demo.Data.Repositories;

public class UserRepository(DemoContext demoContext, ILogger<UserRepository> logger)
{
    #region Private Fields

    private readonly DemoContext _context = demoContext;
    private readonly ILogger<UserRepository> _logger = logger;

    #endregion Private Fields

    #region Public Methods

    public async Task AddUserAsync(User user)
    {
        if (user.Error == ErrorType.Validation)
        {
            _logger.LogError("Validation error for user with email {EmailAddress}", user.EmailAddress);
            return;
        }
        else if (user.Error == ErrorType.Critical)
        {
            throw new InvalidOperationException("Critical error occurred while adding user");
        }

        var entity = new Entities.User
        {
            EmailAddress = user.EmailAddress
        };

        _context.Users.Add(entity);
        await _context.SaveChangesAsync();
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
         => await _context.Users.Select(entity => new User
         {
             EmailAddress = entity.EmailAddress,
             Error = errorChances != null ? GetRandomErrorType(errorChances) : ErrorType.None
         }).ToListAsync();

    #endregion Public Methods

    #region Private Methods

    private ErrorType GetRandomErrorType(IDictionary<ErrorType, decimal> errorChances)
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