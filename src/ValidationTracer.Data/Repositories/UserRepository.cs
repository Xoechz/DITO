using Microsoft.EntityFrameworkCore;
using ValidationTracer.Data.Models;

namespace ValidationTracer.Data.Repositories;

public class UserRepository(ValidationTracerContext validationTracerContext)
{
    #region Private Fields

    private readonly ValidationTracerContext _context = validationTracerContext;

    #endregion Private Fields

    #region Public Methods

    public async Task AddUserAsync(User user)
    {
        var entity = new Entities.User
        {
            EmailAddress = user.EmailAddress,
            InternalProperty = user.InternalProperty,
            ExternalApiProperty = user.ExternalApiProperty,
            ExternalDbProperty = user.ExternalDbProperty,
            CostCenterCode = user.CostCenterCode
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
    }

    public async Task<IEnumerable<User>> GetUsersAsync()
         => await _context.Users.Select(entity => new User
         {
             EmailAddress = entity.EmailAddress,
             InternalProperty = entity.InternalProperty,
             ExternalApiProperty = entity.ExternalApiProperty,
             ExternalDbProperty = entity.ExternalDbProperty,
             CostCenterCode = entity.CostCenterCode
         }).ToListAsync();

    public async Task UpdateUserAsync(User user)
    {
        var entity = await _context.Users.FirstOrDefaultAsync(user => user.EmailAddress == user.EmailAddress)
            ?? throw new KeyNotFoundException($"User with email address {user.EmailAddress} not found");

        entity.InternalProperty = user.InternalProperty;
        entity.ExternalApiProperty = user.ExternalApiProperty;
        entity.ExternalDbProperty = user.ExternalDbProperty;
        entity.CostCenterCode = user.CostCenterCode;

        _context.Users.Update(entity);
        await _context.SaveChangesAsync();
    }

    #endregion Public Methods
}