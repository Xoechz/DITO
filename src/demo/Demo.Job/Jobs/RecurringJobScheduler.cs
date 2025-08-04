using Hangfire;
using Hangfire.Common;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;

namespace Demo.Common.Jobs;

public class RecurringJobScheduler(ILogger<RecurringJobScheduler> logger,
                                   IRecurringJobManager recurringJobManager,
                                   IServiceScopeFactory serviceScopeFactory)
{
    #region Private Fields

    private readonly ILogger<RecurringJobScheduler> _logger = logger;
    private readonly IRecurringJobManager _recurringJobManager = recurringJobManager;
    private readonly IServiceScopeFactory _serviceScopeFactory = serviceScopeFactory;

    #endregion Private Fields

    #region Public Methods

    public void ScheduleRecurringJobs()
    {
        using var scope = _serviceScopeFactory.CreateScope();

        var jobs = scope.ServiceProvider.GetServices<IRecurringJob>();
        int count = jobs.Count();
        _logger.LogInformation("Scheduling recurring {count} jobs", count);

        foreach (var job in jobs)
        {
            var jobName = job.GetType().FullName;
            _recurringJobManager.AddOrUpdate(jobName, GetHangfireJob(job), job.CronExpression);
            _logger.LogInformation("Scheduled job {JobId} with schedule {Schedule}", jobName, job.CronExpression);
        }

        _logger.LogInformation("Scheduled {count} recurring jobs", count);
    }

    #endregion Public Methods

    #region Private Methods

    private Job GetHangfireJob(IRecurringJob recurringJob)
    {
        var jobType = recurringJob.GetType();
        var jobMethod = jobType.GetMethod(nameof(IRecurringJob.DoWork));

        return new Job(jobMethod, jobType);
    }

    #endregion Private Methods
}