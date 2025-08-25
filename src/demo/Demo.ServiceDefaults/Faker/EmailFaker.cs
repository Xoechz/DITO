using Bogus;

namespace Demo.ServiceDefaults.Faker;

public class EmailFaker : Faker<string>
{
    #region Public Constructors

    public EmailFaker()
    {
        ConfigureEmailFaker(1);
    }

    private void ConfigureEmailFaker(int seed)
    {
        UseSeed(seed)
            .CustomInstantiator(f => f.Internet.Email());
    }

    #endregion Public Constructors
}