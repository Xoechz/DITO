var builder = DistributedApplication.CreateBuilder(args);

var sql = builder.AddSqlServer("sql")
    .WithLifetime(ContainerLifetime.Persistent)
    .WithDataVolume();

var externalDb = sql.AddDatabase("external");
var demoDb = sql.AddDatabase("demo");

var migration = builder.AddProject<Projects.Demo_MigrationService>("migrations")
    .WithReference(externalDb)
    .WithReference(demoDb)
    .WaitFor(externalDb)
    .WaitFor(demoDb);

builder.AddProject<Projects.Demo_ApiService>("apiservice")
    .WithReference(demoDb)
    .WaitForCompletion(migration);

builder.AddProject<Projects.External_ApiService>("external-apiservice")
    .WaitForCompletion(migration);

builder.AddProject<Projects.Demo_JobService>("jobservice")
    .WithReference(externalDb)
    .WaitForCompletion(migration);

builder.Build().Run();