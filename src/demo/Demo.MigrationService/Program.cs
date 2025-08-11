using Demo.Data;
using Demo.MigrationService;
using Demo.MigrationService.Faker;
using Demo.ServiceDefaults;

var builder = Host.CreateApplicationBuilder(args);
builder.AddServiceDefaults();
builder.Services.AddHostedService<Worker>();

var serviceName = builder.Configuration["SERVICE_NAME"]
    ?? throw new InvalidOperationException();

builder.AddSqlServerDbContext<DemoContext>("DB-" + serviceName);

builder.Services.AddTransient<UserFaker>();

var host = builder.Build();
host.Run();