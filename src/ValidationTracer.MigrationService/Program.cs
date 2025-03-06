using External.Data;
using ValidationTracer.Data;
using ValidationTracer.MigrationService;
using ValidationTracer.ServiceDefaults;

var builder = Host.CreateApplicationBuilder(args);
builder.AddServiceDefaults();
builder.Services.AddHostedService<Worker>();

builder.Services.AddOpenTelemetry()
    .WithTracing(tracing => tracing.AddSource(Worker.ActivitySourceName));

builder.AddSqlServerDbContext<ExternalContext>("external");
builder.AddSqlServerDbContext<ValidationTracerContext>("validationTracer");

var host = builder.Build();
host.Run();