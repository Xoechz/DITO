using Bogus;

namespace Demo.ServiceDefaults.Faker;

public class EmailFaker : Faker<string>
{
    #region Public Constructors

    public EmailFaker()
    {
        UseSeed(1)
            .CustomInstantiator(f => f.Internet.Email());
    }

    #endregion Public Constructors
}