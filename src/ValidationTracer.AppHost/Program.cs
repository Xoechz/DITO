var builder = DistributedApplication.CreateBuilder(args);

builder.AddProject<Projects.ValidationTracer_MigrationService>("migrations");

builder.AddProject<Projects.ValidationTracer_ApiService>("apiservice");

builder.AddProject<Projects.External_ApiService>("external-apiservice");

builder.Build().Run();