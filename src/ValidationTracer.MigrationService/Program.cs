using External.Data;
using ValidationTracer.Data;
using ValidationTracer.MigrationService;
using ValidationTracer.MigrationService.Faker;
using ValidationTracer.ServiceDefaults;

var builder = Host.CreateApplicationBuilder(args);
builder.AddServiceDefaults();
builder.Services.AddHostedService<Worker>();

builder.Services.AddOpenTelemetry()
    .WithTracing(tracing => tracing.AddSource(Worker.ActivitySourceName));

builder.AddSqlServerDbContext<ExternalContext>("external");
builder.AddSqlServerDbContext<ValidationTracerContext>("validationTracer");

builder.Services.AddTransient<CostCenterFaker>();
builder.Services.AddTransient<UserFaker>();
builder.Services.AddTransient<ExternalDbUserFaker>();

var host = builder.Build();
host.Run();