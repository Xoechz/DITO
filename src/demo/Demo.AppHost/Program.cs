var builder = DistributedApplication.CreateBuilder(args);

var sql = builder.AddSqlServer("sql")
    .WithLifetime(ContainerLifetime.Persistent)
    .WithDataVolume();

var services = Enumerable.Range(0, 3);
var urls = string.Join(",", services.Select(i => "http://127.0.0.1:" + (5280 + i)));

foreach (var serviceIndex in services)
{
    var serviceName = "Service-" + serviceIndex;
    var demoDb = sql.AddDatabase("DB-" + serviceIndex);

    var migration = builder.AddProject<Projects.Demo_MigrationService>("Migration-" + serviceIndex)
        .WithEnvironment("SERVICE_NAME", "Migration-" + serviceIndex)
        .WithEnvironment("SERVICE_INDEX", serviceIndex.ToString())
        .WithEnvironment("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
        .WithReference(demoDb)
        .WaitFor(demoDb);

    builder.AddProject<Projects.Demo_JobService>(serviceName)
        .WithEnvironment("SERVICE_NAME", serviceName)
        .WithEnvironment("SERVICE_INDEX", serviceIndex.ToString())
        .WithEnvironment("TARGET_URLS", urls)
        .WithEnvironment("CRON_EXPRESSION", "*/" + (serviceIndex + 1) + " * * * *")
        .WithEnvironment("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
        .WithHttpEndpoint(5280 + serviceIndex)
        .WithHttpsEndpoint(7160 + serviceIndex)
        .WithReference(demoDb)
        .WaitForCompletion(migration);
}

await builder.Build().RunAsync();