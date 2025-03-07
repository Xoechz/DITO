using Bogus;
using ValidationTracer.Data.Entities;
using ValidationTracer.ServiceDefaults.Faker;

namespace ValidationTracer.MigrationService.Faker;

public class UserFaker : Faker<User>
{
    public UserFaker(EmailFaker emailFaker,
                     CostCenterFaker costCenterFaker)
    {
        var emails = emailFaker.Cache;
        var costCenters = costCenterFaker.Generate(100);
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
}