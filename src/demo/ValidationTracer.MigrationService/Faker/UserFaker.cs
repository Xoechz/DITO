using Bogus;
using ValidationTracer.Common.Fakers;
using ValidationTracer.Data.Entities;

namespace ValidationTracer.MigrationService.Faker;

public class UserFaker : CachedFakerBase<User>
{
    #region Public Constructors

    public UserFaker(EmailFaker emailFaker,
                     CostCenterFaker costCenterFaker)
    {
        var emails = emailFaker.Cache;
        var costCenters = costCenterFaker.Cache;
        UseSeed(1)
            .RuleFor(u => u.EmailAddress, f =>
            {
                var email = f.PickRandom(emails);
                emails.Remove(email);
                return email;
            })
            .RuleFor(u => u.InternalProperty, f => f.Random.AlphaNumeric(10))
            .RuleFor(u => u.CostCenterCode, f => f.PickRandom(costCenters).OrNull(f)?.Code);
    }

    #endregion Public Constructors

    #region Public Properties

    public override int CacheSize => 3000;

    #endregion Public Properties
}