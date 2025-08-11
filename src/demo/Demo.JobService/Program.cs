using Demo.Common.Jobs;
using Demo.Data;
using Demo.Data.Repositories;
using Demo.JobService.Jobs;
using Demo.ServiceDefaults;
using Hangfire;

var builder = WebApplication.CreateBuilder(args);

builder.AddServiceDefaults();

var serviceName = builder.Configuration["SERVICE_NAME"]
    ?? throw new InvalidOperationException();

var connectionString = builder.Configuration.GetConnectionString("DB-" + serviceName)
    ?? throw new InvalidOperationException("Connection String DB-" + serviceName + " is not configured.");

builder.AddSqlServerDbContext<DemoContext>("DB-" + serviceName);

builder.Services.AddHangfireServer()
    .AddSingleton<RecurringJobScheduler>()
    .AddTransient<JobWorker>()
    .AddHangfire(opt => opt.UseSqlServerStorage(connectionString));

builder.Services.AddControllers();
builder.Services.AddSwaggerGen();

builder.Services.AddScoped<UserRepository>();

var app = builder.Build();

app.MapDefaultEndpoints();
app.UseHttpsRedirection();
app.MapControllers();
app.UseSwagger();
app.UseSwaggerUI();
app.MapHangfireDashboard();

var scheduler = app.Services.GetRequiredService<RecurringJobScheduler>();
scheduler.ScheduleRecurringJobs();

app.Run();