using Bogus;
using External.ApiService.Models;
using ValidationTracer.Common.Fakers;

namespace External.ApiService.Faker;

public class ExternalApiUserFaker : Faker<ExternalApiUser>
{
    public ExternalApiUserFaker(EmailFaker emailFaker)
    {
        var emails = emailFaker.Cache;
        UseSeed(1)
            .RuleFor(u => u.EmailAddress, f =>
            {
                var email = f.PickRandom(emails);
                emails.Remove(email);
                return email;
            })
            .RuleFor(u => u.ExternalApiProperty, f => f.Lorem.Word())
            .RuleFor(u => u.CostCenterCode, f => f.Random.Number(0, 9999).ToString().PadLeft(4, '0'));
    }
}