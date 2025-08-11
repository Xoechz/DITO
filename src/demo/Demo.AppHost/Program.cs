var builder = DistributedApplication.CreateBuilder(args);

var sql = builder.AddSqlServer("sql")
    .WithLifetime(ContainerLifetime.Persistent)
    .WithDataVolume();

List<string> serviceNames = ["Service0", "Service1", "Service2", "Service3", "Service4", "Service5", "Service6", "Service7", "Service8", "Service9"];

foreach (var serviceName in serviceNames)
{
    var demoDb = sql.AddDatabase("DB-" + serviceName);

    var migration = builder.AddProject<Projects.Demo_MigrationService>("Migration-" + serviceName)
        .WithEnvironment("SERVICE_NAME", serviceName)
        .WithReference(demoDb)
        .WaitFor(demoDb);

    builder.AddProject<Projects.Demo_JobService>(serviceName)
        .WithEnvironment("SERVICE_NAME", serviceName)
        .WithReference(demoDb)
        .WaitForCompletion(migration);
}

builder.Build().Run();