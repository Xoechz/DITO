using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Design;

namespace External.Data;

internal class DesignTimeExternalContextFactory : IDesignTimeDbContextFactory<ExternalContext>
{
    public ExternalContext CreateDbContext(string[] args)
    {
        var optionsBuilder = new DbContextOptionsBuilder<ExternalContext>();
        optionsBuilder.UseSqlServer("Server=.;Database=External;Trusted_Connection=True;");
        return new ExternalContext(optionsBuilder.Options);
    }
}