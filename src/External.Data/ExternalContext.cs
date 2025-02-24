using External.Data.Entities;
using Microsoft.EntityFrameworkCore;

namespace External.Data;

public class ExternalContext(DbContextOptions<ExternalContext> options) : DbContext(options)
{
    public DbSet<ExternalUser> ExternalUsers { get; set; }
}