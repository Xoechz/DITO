using External.Data;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Infrastructure;
using Microsoft.EntityFrameworkCore.Storage;
using System.Diagnostics;
using Demo.Data;
using Demo.MigrationService.Faker;

namespace Demo.MigrationService;

public class Worker(IServiceProvider serviceProvider,
                    IHostApplicationLifetime hostApplicationLifetime,
                    CostCenterFaker costCenterFaker,
                    UserFaker userFaker,
                    ExternalDbUserFaker externalDbUserFaker)
    : BackgroundService
{
    #region Public Fields

    public const string ActivitySourceName = "Migrations";

    #endregion Public Fields

    #region Private Fields

    private static readonly ActivitySource s_activitySource = new(ActivitySourceName);
    private readonly CostCenterFaker _costCenterFaker = costCenterFaker;
    private readonly ExternalDbUserFaker _externalDbUserFaker = externalDbUserFaker;
    private readonly IHostApplicationLifetime _hostApplicationLifetime = hostApplicationLifetime;
    private readonly IServiceProvider _serviceProvider = serviceProvider;
    private readonly UserFaker _userFaker = userFaker;

    #endregion Private Fields

    #region Protected Methods

    protected override async Task ExecuteAsync(CancellationToken cancellationToken)
    {
        using var activity = s_activitySource.StartActivity("Migrating database", ActivityKind.Client);

        try
        {
            using var scope = _serviceProvider.CreateScope();
            var demoContext = scope.ServiceProvider.GetRequiredService<DemoContext>();
            var externalContext = scope.ServiceProvider.GetRequiredService<ExternalContext>();

            await EnsureDatabaseAsync(demoContext, cancellationToken);
            await RunMigrationAsync(demoContext, cancellationToken);
            await SeedDataAsync(demoContext, cancellationToken);

            await EnsureDatabaseAsync(externalContext, cancellationToken);
            await RunMigrationAsync(externalContext, cancellationToken);
            await SeedDataAsync(externalContext, cancellationToken);
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
        var costCenters = _costCenterFaker.Cache;
        var users = _userFaker.Cache;

        var strategy = dbContext.Database.CreateExecutionStrategy();
        await strategy.ExecuteAsync(async () =>
        {
            // Seed the database
            await using var transaction = await dbContext.Database.BeginTransactionAsync(cancellationToken);
            dbContext.Users.RemoveRange(dbContext.Users);
            dbContext.CostCenters.RemoveRange(dbContext.CostCenters);
            await dbContext.CostCenters.AddRangeAsync(costCenters, cancellationToken);
            await dbContext.Users.AddRangeAsync(users, cancellationToken);
            await dbContext.SaveChangesAsync(cancellationToken);
            await transaction.CommitAsync(cancellationToken);
        });
    }

    private async Task SeedDataAsync(ExternalContext dbContext, CancellationToken cancellationToken)
    {
        var users = _externalDbUserFaker.Cache;
        var strategy = dbContext.Database.CreateExecutionStrategy();
        await strategy.ExecuteAsync(async () =>
        {
            // Seed the database
            await using var transaction = await dbContext.Database.BeginTransactionAsync(cancellationToken);
            dbContext.ExternalUsers.RemoveRange(dbContext.ExternalUsers);
            await dbContext.ExternalUsers.AddRangeAsync(users, cancellationToken);
            await dbContext.SaveChangesAsync(cancellationToken);
            await transaction.CommitAsync(cancellationToken);
        });
    }

    #endregion Private Methods
}