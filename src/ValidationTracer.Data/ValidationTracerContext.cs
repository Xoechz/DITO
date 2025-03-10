using Microsoft.EntityFrameworkCore;
using ValidationTracer.Data.Entities;

namespace ValidationTracer.Data;

public class ValidationTracerContext(DbContextOptions<ValidationTracerContext> options) : DbContext(options)
{
    #region Public Properties

    public DbSet<CostCenter> CostCenters { get; set; }

    public DbSet<User> Users { get; set; }

    #endregion Public Properties
}