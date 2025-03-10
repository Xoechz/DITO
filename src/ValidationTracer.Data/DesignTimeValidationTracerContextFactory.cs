using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Design;

namespace ValidationTracer.Data;

internal class DesignTimeValidationTracerContextFactory : IDesignTimeDbContextFactory<ValidationTracerContext>
{
    #region Public Methods

    public ValidationTracerContext CreateDbContext(string[] args)
    {
        var optionsBuilder = new DbContextOptionsBuilder<ValidationTracerContext>();
        optionsBuilder.UseSqlServer("Server=.;Database=External;Trusted_Connection=True;");
        return new ValidationTracerContext(optionsBuilder.Options);
    }

    #endregion Public Methods
}