using Demo.Data.Entities;
using Microsoft.EntityFrameworkCore;

namespace Demo.Data;

public class DemoContext(DbContextOptions<DemoContext> options) : DbContext(options)
{
    #region Public Properties

    public DbSet<User> Users { get; set; }

    #endregion Public Properties
}