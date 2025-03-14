using Hangfire;
using Hangfire.SqlServer;
using ValidationTracer.ApiService.Jobs;
using ValidationTracer.Common.Jobs;
using ValidationTracer.Data;
using ValidationTracer.Data.Repositories;
using ValidationTracer.ServiceDefaults;

var builder = WebApplication.CreateBuilder(args);

builder.AddServiceDefaults();

builder.AddSqlServerDbContext<ValidationTracerContext>("validationTracer");

// Add services to the container.
builder.Services.AddControllers();
// Learn more about configuring OpenAPI at https://aka.ms/aspnet/openapi
builder.Services.AddSwaggerGen();
builder.Services.AddHangfireServer()
    .AddSingleton<RecurringJobScheduler>()
    .AddTransient<IRecurringJob, ExternalApiCollector>()
    .AddHangfire(opt =>
    {
        opt.UseSqlServerStorage(builder.Configuration.GetConnectionString("validationTracer"),
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

builder.Services.AddScoped<UserRepository>();
builder.Services.AddScoped<CostCenterRepository>();

var app = builder.Build();

app.UseHttpsRedirection();

app.UseAuthorization();

app.MapControllers();
app.UseSwagger();
app.UseSwaggerUI();
app.MapHangfireDashboard();

var scheduler = app.Services.GetRequiredService<RecurringJobScheduler>();
scheduler.ScheduleRecurringJobs();

app.Run();