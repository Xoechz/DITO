using Bogus;
using Demo.Data.Models;

namespace Demo.MigrationService.Faker;

public class UserFaker : Faker<User>
{
    #region Public Constructors

    public UserFaker()
    {
        UseSeed(1)
            .RuleFor(u => u.EmailAddress, f => f.Internet.Email())
            .RuleFor(u => u.Error, f => f.PickRandom<ErrorType>());
    }

    #endregion Public Constructors
}