var builder = DistributedApplication.CreateBuilder(args);

var sql = builder.AddSqlServer("sql");
var externalDb = sql.AddDatabase("external");
var validationTracerDb = sql.AddDatabase("validationTracer");

var migration = builder.AddProject<Projects.ValidationTracer_MigrationService>("migrations")
    .WithReference(externalDb)
    .WithReference(validationTracerDb)
    .WaitFor(externalDb)
    .WaitFor(validationTracerDb);

builder.AddProject<Projects.ValidationTracer_ApiService>("apiservice")
    .WithReference(validationTracerDb)
    .WaitFor(migration);

builder.AddProject<Projects.External_ApiService>("external-apiservice")
    .WaitFor(migration);

builder.Build().Run();