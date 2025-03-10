using Bogus;

namespace ValidationTracer.Common.Fakers;

public class EmailFaker : Faker<string>
{
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

        Cache = Generate(10000);
    }

    public List<string> Cache { get; }
}