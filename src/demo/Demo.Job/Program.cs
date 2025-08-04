using Hangfire;
using Hangfire.SqlServer;
using Demo.Common.Jobs;
using Demo.JobService.Jobs;
using Demo.ServiceDefaults;

var builder = WebApplication.CreateBuilder(args);

builder.AddServiceDefaults();

builder.AddSqlServerDbContext<DemoContext>("demo");

builder.Services.AddHangfireServer()
    .AddSingleton<RecurringJobScheduler>()
    .AddTransient<JobWorker>()
    .AddHangfire();

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