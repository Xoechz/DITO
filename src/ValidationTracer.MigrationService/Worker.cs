using External.Data;
using External.Data.Entities;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Infrastructure;
using Microsoft.EntityFrameworkCore.Storage;
using System.Diagnostics;
using ValidationTracer.Data;
using ValidationTracer.Data.Entities;

namespace ValidationTracer.MigrationService;

public class Worker(
    IServiceProvider serviceProvider,
    IHostApplicationLifetime hostApplicationLifetime) : BackgroundService
{
    public const string ActivitySourceName = "Migrations";
    private static readonly ActivitySource s_activitySource = new(ActivitySourceName);

    protected override async Task ExecuteAsync(CancellationToken cancellationToken)
    {
        using var activity = s_activitySource.StartActivity("Migrating database", ActivityKind.Client);

        try
        {
            using var scope = serviceProvider.CreateScope();
            var validationTracerContext = scope.ServiceProvider.GetRequiredService<ValidationTracerContext>();
            var externalContext = scope.ServiceProvider.GetRequiredService<ExternalContext>();

            await EnsureDatabaseAsync(validationTracerContext, cancellationToken);
            await RunMigrationAsync(validationTracerContext, cancellationToken);
            await SeedDataAsync(validationTracerContext, cancellationToken);

            await EnsureDatabaseAsync(externalContext, cancellationToken);
            await RunMigrationAsync(externalContext, cancellationToken);
            await SeedDataAsync(externalContext, cancellationToken);
        }
        catch (Exception ex)
        {
            activity?.AddException(ex);
            throw;
        }

        hostApplicationLifetime.StopApplication();
    }

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

    private static async Task SeedDataAsync(ValidationTracerContext dbContext, CancellationToken cancellationToken)
    {
        CostCenter c = new()
        {
            Code = "1"
        };

        User u = new()
        {
            EmailAddress = "1",
            ExternalDbProperty = "1",
            ExternalApiProperty = "1",
            InternalProperty = "1",
            CostCenterCode = "1"
        };

        var strategy = dbContext.Database.CreateExecutionStrategy();
        await strategy.ExecuteAsync(async () =>
        {
            // Seed the database
            await using var transaction = await dbContext.Database.BeginTransactionAsync(cancellationToken);
            await dbContext.CostCenters.AddAsync(c, cancellationToken);
            await dbContext.Users.AddAsync(u, cancellationToken);
            await dbContext.SaveChangesAsync(cancellationToken);
            await transaction.CommitAsync(cancellationToken);
        });
    }

    private static async Task SeedDataAsync(ExternalContext dbContext, CancellationToken cancellationToken)
    {
        ExternalDbUser e = new()
        {
            EmailAddress = "1",
            ExternalDbProperty = "1",
            CostCenterCode = "1"
        };

        var strategy = dbContext.Database.CreateExecutionStrategy();
        await strategy.ExecuteAsync(async () =>
        {
            // Seed the database
            await using var transaction = await dbContext.Database.BeginTransactionAsync(cancellationToken);
            await dbContext.ExternalUsers.AddAsync(e, cancellationToken);
            await dbContext.SaveChangesAsync(cancellationToken);
            await transaction.CommitAsync(cancellationToken);
        });
    }
}