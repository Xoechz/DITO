using Microsoft.EntityFrameworkCore;
using Demo.Data.Entities;

namespace Demo.Data;

public class DemoContext(DbContextOptions<DemoContext> options) : DbContext(options)
{
    #region Public Properties

    public DbSet<User> Users { get; set; }

    #endregion Public Properties
}