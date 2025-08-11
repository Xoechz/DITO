using Demo.OpenTelemetry.Extensions;
using Demo.OpenTelemetry.Jobs.Filters;
using Hangfire;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

namespace Demo.OpenTelemetry.Jobs.Extensions;

/// <summary>
/// Extension methods for configuring OpenTelemetry with hangfire in the service collection.
/// </summary>
public static class ServiceCollectionExtensions
{
    #region Public Methods

    /// <summary>
    /// Configures OpenTelemetry and adds Hangfire failure tracing for OpenTelemetry.
    /// </summary>
    /// <param name="services"><see cref="IServiceCollection"/></param>
    /// <param name="configuration"><see cref="IConfiguration"/></param>
    /// <param name="environment"><see cref="IHostEnvironment"/></param>
    /// <returns><see cref="IServiceCollection"/> for method chaining</returns>
    public static IServiceCollection ConfigureOpenTelemetryWithHangfire(this IServiceCollection services, IConfiguration configuration, IHostEnvironment environment)
    {
        services.ConfigureOpenTelemetry(configuration, environment, out var activitySource);
        // Adds the Filter to every Hangfire job to trace failures with OpenTelemetry
        GlobalJobFilters.Filters.Add(new OpenTelemetryHangfireFilter(activitySource));

        return services;
    }

    #endregion Public Methods
}