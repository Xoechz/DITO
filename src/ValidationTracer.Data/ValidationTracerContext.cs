using Microsoft.EntityFrameworkCore;
using ValidationTracer.Data.Entities;

namespace ValidationTracer.Data;

public class ValidationTracerContext(DbContextOptions<ValidationTracerContext> options) : DbContext(options)
{
    public DbSet<User> Users { get; set; }

    public DbSet<CostCenter> CostCenters { get; set; }
}