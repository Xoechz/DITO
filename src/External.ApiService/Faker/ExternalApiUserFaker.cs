using External.ApiService.Models;
using ValidationTracer.Common.Fakers;

namespace External.ApiService.Faker;

public class ExternalApiUserFaker : CachedFakerBase<ExternalApiUser>
{
    #region Public Constructors

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
            .RuleFor(u => u.ExternalApiProperty, f =>
            {
                var word = f.Lorem.Word();
                return word.Length > 10 ? word[..10] : word;
            })
            .RuleFor(u => u.CostCenterCode, f => f.Random.Number(0, 9999).ToString().PadLeft(4, '0'));
    }

    #endregion Public Constructors

    #region Public Properties

    public override int CacheSize => 1000;

    #endregion Public Properties
}