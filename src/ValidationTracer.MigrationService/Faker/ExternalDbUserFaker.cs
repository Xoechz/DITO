using Bogus;
using External.Data.Entities;
using ValidationTracer.ServiceDefaults.Faker;

namespace ValidationTracer.MigrationService.Faker;

public class ExternalDbUserFaker : Faker<ExternalDbUser>
{
    public ExternalDbUserFaker(EmailFaker emailFaker)
    {
        var emails = emailFaker.Cache;
        UseSeed(1)
            .RuleFor(u => u.EmailAddress, f =>
            {
                var email = f.PickRandom(emails);
                emails.Remove(email);
                return email;
            })
            .RuleFor(u => u.ExternalDbProperty, f => f.Commerce.Product());
    }
}