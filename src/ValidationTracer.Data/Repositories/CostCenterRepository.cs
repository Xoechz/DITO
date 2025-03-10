using Microsoft.EntityFrameworkCore;
using ValidationTracer.Data.Models;

namespace ValidationTracer.Data.Repositories;

public class CostCenterRepository(ValidationTracerContext validationTracerContext)
{
    #region Private Fields

    private readonly ValidationTracerContext _context = validationTracerContext;

    #endregion Private Fields

    #region Public Methods

    public async Task<IEnumerable<CostCenter>> GetCostCentersAsync()
    {
        var entities = await _context.CostCenters.ToListAsync();
        return entities.Select(entity => new CostCenter
        {
            Code = entity.Code
        });
    }

    #endregion Public Methods
}