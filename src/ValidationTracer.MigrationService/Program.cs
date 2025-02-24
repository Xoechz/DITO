using External.Data;
using Microsoft.EntityFrameworkCore;
using ValidationTracer.Data;
using ValidationTracer.MigrationService;
using ValidationTracer.ServiceDefaults;

var builder = Host.CreateApplicationBuilder(args);
builder.AddServiceDefaults();
builder.Services.AddHostedService<Worker>();

builder.Services.AddOpenTelemetry()
    .WithTracing(tracing => tracing.AddSource(Worker.ActivitySourceName));

builder.Services.AddDbContext<ExternalContext>(options =>
    {
        var path = Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData);
        options.UseSqlite($"Data Source={Path.Join(path, "external.db")}");
    });

builder.Services.AddDbContext<ValidationTracerContext>(options =>
    {
        var path = Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData);
        options.UseSqlite($"Data Source={Path.Join(path, "validationTracer.db")}");
    });

var host = builder.Build();
host.Run();