using Bogus;
using ValidationTracer.Data.Entities;

namespace ValidationTracer.MigrationService.Faker;

public class CostCenterFaker : Faker<CostCenter>
{
    public CostCenterFaker()
    {
        List<string> existing = [];

        UseSeed(1)
            .RuleFor(c => c.Code, f =>
            {
                var costCenter = f.Random.Number(0, 9999).ToString().PadLeft(4, '0');
                while (existing.Contains(costCenter))
                {
                    costCenter = f.Random.Number(0, 9999).ToString().PadLeft(4, '0');
                }
                existing.Add(costCenter);
                return costCenter;
            });
    }
}