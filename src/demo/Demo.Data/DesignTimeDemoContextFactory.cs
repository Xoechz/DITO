using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Design;

namespace Demo.Data;

internal class DesignTimeDemoContextFactory : IDesignTimeDbContextFactory<DemoContext>
{
    #region Public Methods

    public DemoContext CreateDbContext(string[] args)
    {
        var optionsBuilder = new DbContextOptionsBuilder<DemoContext>();
        optionsBuilder.UseSqlServer("Server=.;Database=DesignTime;Trusted_Connection=True;");
        return new DemoContext(optionsBuilder.Options);
    }

    #endregion Public Methods
}