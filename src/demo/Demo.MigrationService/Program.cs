using Demo.Data;
using Demo.MigrationService;
using Demo.ServiceDefaults;

var builder = Host.CreateApplicationBuilder(args);
builder.AddServiceDefaults();
builder.Services.AddHostedService<Worker>();

var serviceIndex = builder.Configuration["SERVICE_INDEX"]
    ?? throw new InvalidOperationException();

builder.AddSqlServerDbContext<DemoContext>("DB-" + serviceIndex);

var host = builder.Build();
host.Run();