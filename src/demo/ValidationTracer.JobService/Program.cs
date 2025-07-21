using External.Data;
using Hangfire;
using Hangfire.SqlServer;
using ValidationTracer.Common.Jobs;
using ValidationTracer.JobService.Jobs;
using ValidationTracer.ServiceDefaults;

var builder = WebApplication.CreateBuilder(args);

builder.AddServiceDefaults();

builder.AddSqlServerDbContext<ExternalContext>("external");

builder.Services.AddHangfireServer()
    .AddSingleton<RecurringJobScheduler>()
    .AddTransient<IRecurringJob, ExportWorker>()
    .AddHangfire(opt =>
    {
        opt.UseSqlServerStorage(builder.Configuration.GetConnectionString("external"),
            new SqlServerStorageOptions
            {
                CommandBatchMaxTimeout = TimeSpan.FromHours(1),
                SlidingInvisibilityTimeout = TimeSpan.FromMinutes(15),
                QueuePollInterval = TimeSpan.FromMinutes(1),
                UseRecommendedIsolationLevel = true,
                DisableGlobalLocks = true,
                SchemaName = "hangfire"
            })
        .WithJobExpirationTimeout(TimeSpan.FromDays(31));
    });

var app = builder.Build();

app.MapDefaultEndpoints();

// Configure the HTTP request pipeline.

app.UseHttpsRedirection();
app.MapHangfireDashboard();

var scheduler = app.Services.GetRequiredService<RecurringJobScheduler>();
scheduler.ScheduleRecurringJobs();

app.Run();