namespace ValidationTracer.Common.Fakers;

public class EmailFaker : CachedFakerBase<string>
{
    #region Public Constructors

    public EmailFaker()
    {
        List<string> existing = [];
        UseSeed(1)
            .CustomInstantiator(f =>
            {
                var counter = 0;
                var email = f.Internet.Email();

                while (existing.Contains(email) && counter < 1000)
                {
                    email = f.Internet.Email();
                    counter++;
                }

                if (counter >= 1000)
                {
                    email = $"{f.Random.AlphaNumeric(10)}@{f.Lorem.Word()}.{f.Lorem.Word()}";
                }

                existing.Add(email);
                return email;
            });
    }

    #endregion Public Constructors

    #region Public Properties

    public override int CacheSize => 10000;

    #endregion Public Properties
}