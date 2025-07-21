using ValidationTracer.Common.Fakers;
using ValidationTracer.Data.Entities;

namespace ValidationTracer.MigrationService.Faker;

public class CostCenterFaker : CachedFakerBase<CostCenter>
{
    #region Public Constructors

    public CostCenterFaker()
    {
        List<int> existing = [];

        UseSeed(1)
            .RuleFor(c => c.Code, f =>
            {
                if (existing.Count >= 10000)
                {
                    throw new InvalidOperationException("Cannot generate more than 10000 unique cost centers");
                }

                var costCenter = f.Random.Number(0, 9999);

                while (existing.Contains(costCenter))
                {
                    costCenter = (costCenter + 1) % 10000;
                }

                existing.Add(costCenter);
                return costCenter.ToString().PadLeft(4, '0');
            });
    }

    #endregion Public Constructors

    #region Public Properties

    public override int CacheSize => 5000;

    #endregion Public Properties
}