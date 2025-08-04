using External.Data;
using Demo.Data;
using Demo.MigrationService;
using Demo.MigrationService.Faker;
using Demo.ServiceDefaults;

var builder = Host.CreateApplicationBuilder(args);
builder.AddServiceDefaults();
builder.Services.AddHostedService<Worker>();

builder.Services.AddOpenTelemetry()
    .WithTracing(tracing => tracing.AddSource(Worker.ActivitySourceName));

builder.AddSqlServerDbContext<ExternalContext>("external");
builder.AddSqlServerDbContext<DemoContext>("demo");

builder.Services.AddTransient<CostCenterFaker>();
builder.Services.AddTransient<UserFaker>();
builder.Services.AddTransient<ExternalDbUserFaker>();

var host = builder.Build();
host.Run();