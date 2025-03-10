using External.Data.Entities;
using Microsoft.EntityFrameworkCore;

namespace External.Data;

public class ExternalContext(DbContextOptions<ExternalContext> options) : DbContext(options)
{
    #region Public Properties

    public DbSet<ExternalDbUser> ExternalUsers { get; set; }

    #endregion Public Properties
}