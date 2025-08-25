using Azure.Monitor.OpenTelemetry.Exporter;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using OpenTelemetry;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;
using System.Diagnostics;

namespace Demo.OpenTelemetry.Extensions;

/// <summary>
/// Extension methods for configuring OpenTelemetry in the service collection.
/// </summary>
public static class ServiceCollectionExtension
{
    #region Public Methods

    /// <summary>
    /// Configures basic OpenTelemetry.
    /// </summary>
    /// <param name="services"><see cref="IServiceCollection"/></param>
    /// <param name="configuration"><see cref="IConfiguration"/></param>
    /// <returns><see cref="IServiceCollection"/> for method chaining</returns>
    public static IServiceCollection ConfigureBasicOpenTelemetry(this IServiceCollection services, IConfiguration configuration)
    {
        var otelBuilder = services.AddOpenTelemetry()
            .WithLogging()
            .WithMetrics(metrics =>
            {
                metrics.AddAspNetCoreInstrumentation()
                    .AddHttpClientInstrumentation()
                    .AddSqlClientInstrumentation()
                    .AddRuntimeInstrumentation()
                    .AddProcessInstrumentation();
            })
            .WithTracing(tracing =>
            {
                tracing.AddAspNetCoreInstrumentation()
                    .AddEntityFrameworkCoreInstrumentation()
                    .AddHttpClientInstrumentation();
            });

        // Check if the OTLP exporter or Azure Monitor is configured via environment variables.
        var useOtlpExporter = !string.IsNullOrWhiteSpace(configuration["OTEL_EXPORTER_OTLP_ENDPOINT"]);
        var useAzureMonitor = !string.IsNullOrWhiteSpace(configuration["APPLICATIONINSIGHTS_CONNECTION_STRING"]);

        if (useOtlpExporter)
        {
            // Aspire sets the OTEL_EXPORTER_OTLP_ENDPOINT environment variable to the OpenTelemetry Collector endpoint automatically.
            // This is then only used locally.
            // Without Aspire no local OpenTelemetry Collector is used.
            otelBuilder.UseOtlpExporter();
        }

        if (useAzureMonitor)
        {
            // If the APPLICATIONINSIGHTS_CONNECTION_STRING environment variable is set, the Azure Monitor exporter is used.
            // Application Insights is a subsystem of Azure Monitor.
            otelBuilder.UseAzureMonitorExporter();
        }

        return services;
    }

    /// <summary>
    /// The swietelsky way for OpenTelemetry configuration.
    /// </summary>
    /// <param name="services"><see cref="IServiceCollection"/></param>
    /// <param name="configuration"><see cref="IConfiguration"/></param>
    /// <param name="environment"><see cref="IHostEnvironment"/></param>
    /// <returns><see cref="IServiceCollection"/> for method chaining</returns>
    public static IServiceCollection ConfigureOpenTelemetry(this IServiceCollection services, IConfiguration configuration, IHostEnvironment environment)
        => ConfigureOpenTelemetry(services, configuration, environment, out _);

    /// <summary>
    /// The swietelsky way for OpenTelemetry configuration. Outputs the <see cref="ActivitySource"/> Singleton that the application uses.
    /// </summary>
    /// <param name="services"><see cref="IServiceCollection"/></param>
    /// <param name="configuration"><see cref="IConfiguration"/></param>
    /// <param name="environment"><see cref="IHostEnvironment"/></param>
    /// <param name="activitySource"><see cref="ActivitySource"/> Singleton that the application uses</param>
    /// <returns><see cref="IServiceCollection"/> for method chaining</returns>
    public static IServiceCollection ConfigureOpenTelemetry(this IServiceCollection services, IConfiguration configuration, IHostEnvironment environment, out ActivitySource activitySource)
    {
        var serviceName = configuration["SERVICE_NAME"] ?? environment.ApplicationName;

        // for manual trace instrumentation
        // to use it, inject the ActivitySource into you class and then create an Activity like this:
        // using (var activity = activitySource.StartActivity("MyOperation"))
        // Do not forget the using statement or dispose the activity manually to end the trace.
        activitySource = new ActivitySource(serviceName);
        services.AddSingleton(activitySource);

        // OpenTelemetry configuration
        var otelBuilder = services.AddOpenTelemetry()
            // Add the service name to the resource attributes so application insights displays it correctly in the application map
            .ConfigureResource(rb => rb.AddService(serviceName))
            // Add the logging provider to export logs of the ILogger interface
            .WithLogging()
            // Metrics are used to collect performance data, such as request duration, memory usage, etc.
            .WithMetrics(metrics =>
            {
                // AspNetCoreInstrumentation adds general metrics for ASP.NET Core applications such as request duration, request count, etc.
                metrics.AddAspNetCoreInstrumentation()
                    // HttpClientInstrumentation adds metrics for outgoing HTTP requests made by the application.
                    .AddHttpClientInstrumentation()
                    // SqlClientInstrumentation adds metrics for SQL database operations.
                    .AddSqlClientInstrumentation()
                    // RuntimeInstrumentation adds metrics for the .NET runtime, such as garbage collection, thread pool usage, etc.
                    .AddRuntimeInstrumentation()
                    // ProcessInstrumentation adds metrics for the process, such as CPU usage, memory usage, etc.
                    .AddProcessInstrumentation();
            })
            // Tracing is used to collect detailed information about the execution of requests, jobs and other operations in the application.
            .WithTracing(tracing =>
            {
                // Adds the service name, so the logs can be correlated with traces
                tracing.AddSource(serviceName)
                // AspNetCoreInstrumentation adds tracing for ASP.NET Core operations, such as HTTP requests, middleware, etc.
                    .AddAspNetCoreInstrumentation(o =>
                        {
                            // Exclude Hangfire requests and 401 responses from tracing.
                            // Hangfire requests are made from the Hangfire dashboard and do not need to be traced.
                            // Most 401 responses are caused by NTLM authentication, which is not relevant for tracing.
                            o.Filter = context => !context.Request.Path.ToString().Contains("hangfire")
                                && context.Response.StatusCode != 401;
                        })
                    // EntityFrameworkCoreInstrumentation adds tracing for Entity Framework Core operations, such as database queries and commands.
                    // Node: The SqlClientInstrumentation is not included, because we do not really use raw SQL commands, but hangfire uses them fairly often, which is not relevant.
                    // Also the EntityFramework Traces would be duplicated.
                    // The DB statements are captured to see which SQL commands are executed.
                    .AddEntityFrameworkCoreInstrumentation(o => o.SetDbStatementForText = true)
                    // HttpClientInstrumentation adds tracing for outgoing HTTP requests made by the application.
                    .AddHttpClientInstrumentation();
            });

        // Check if the OTLP exporter or Azure Monitor is configured via environment variables.
        var useOtlpExporter = !string.IsNullOrWhiteSpace(configuration["OTEL_EXPORTER_OTLP_ENDPOINT"]);
        var useAzureMonitor = !string.IsNullOrWhiteSpace(configuration["APPLICATIONINSIGHTS_CONNECTION_STRING"]);

        if (useOtlpExporter)
        {
            // Aspire sets the OTEL_EXPORTER_OTLP_ENDPOINT environment variable to the OpenTelemetry Collector endpoint automatically.
            // This is then only used locally.
            // Without Aspire no local OpenTelemetry Collector is used.
            otelBuilder.UseOtlpExporter();
        }

        if (useAzureMonitor)
        {
            // If the APPLICATIONINSIGHTS_CONNECTION_STRING environment variable is set, the Azure Monitor exporter is used.
            // Application Insights is a subsystem of Azure Monitor.
            otelBuilder.UseAzureMonitorExporter();
        }

        return services;
    }

    #endregion Public Methods
}