using Demo.Data;
using Demo.MigrationService.Faker;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Infrastructure;
using Microsoft.EntityFrameworkCore.Storage;
using System.Diagnostics;

namespace Demo.MigrationService;

public class Worker(IServiceProvider serviceProvider,
                    IHostApplicationLifetime hostApplicationLifetime,
                    UserFaker userFaker,
                    ActivitySource activitySource)
    : BackgroundService
{
    #region Private Fields

    private readonly ActivitySource _activitySource = activitySource;
    private readonly IHostApplicationLifetime _hostApplicationLifetime = hostApplicationLifetime;
    private readonly IServiceProvider _serviceProvider = serviceProvider;
    private readonly UserFaker _userFaker = userFaker;

    #endregion Private Fields

    #region Protected Methods

    protected override async Task ExecuteAsync(CancellationToken cancellationToken)
    {
        using var activity = _activitySource.StartActivity("Migrating database", ActivityKind.Client);

        try
        {
            using var scope = _serviceProvider.CreateScope();
            var demoContext = scope.ServiceProvider.GetRequiredService<DemoContext>();

            await EnsureDatabaseAsync(demoContext, cancellationToken);
            await RunMigrationAsync(demoContext, cancellationToken);
            await SeedDataAsync(demoContext, cancellationToken);
        }
        catch (Exception ex)
        {
            activity?.AddException(ex);
            throw;
        }

        _hostApplicationLifetime.StopApplication();
    }

    #endregion Protected Methods

    #region Private Methods

    private static async Task EnsureDatabaseAsync(DbContext dbContext, CancellationToken cancellationToken)
    {
        var dbCreator = dbContext.GetService<IRelationalDatabaseCreator>();

        var strategy = dbContext.Database.CreateExecutionStrategy();
        await strategy.ExecuteAsync(async () =>
        {
            // Create the database if it does not exist.
            // Do this first so there is then a database to start a transaction against.
            if (!await dbCreator.ExistsAsync(cancellationToken))
            {
                await dbCreator.CreateAsync(cancellationToken);
            }
        });
    }

    private static async Task RunMigrationAsync(DbContext dbContext, CancellationToken cancellationToken)
    {
        var strategy = dbContext.Database.CreateExecutionStrategy();
        await strategy.ExecuteAsync(async () => await dbContext.Database.MigrateAsync(cancellationToken));
    }

    private async Task SeedDataAsync(DemoContext dbContext, CancellationToken cancellationToken)
    {
        var users = _userFaker.Generate(100).Select(u => new Data.Entities.User { EmailAddress = u.EmailAddress });

        var strategy = dbContext.Database.CreateExecutionStrategy();
        await strategy.ExecuteAsync(async () =>
        {
            // Seed the database
            await using var transaction = await dbContext.Database.BeginTransactionAsync(cancellationToken);
            await dbContext.Users.ExecuteDeleteAsync();
            await dbContext.Users.AddRangeAsync(users, cancellationToken);
            await dbContext.SaveChangesAsync(cancellationToken);
            await transaction.CommitAsync(cancellationToken);
        });
    }

    #endregion Private Methods
}