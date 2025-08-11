using Hangfire;

namespace Demo.JobService.Jobs;

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

        var job = scope.ServiceProvider.GetRequiredService<JobWorker>();

        var jobName = job.GetType().FullName;
        _recurringJobManager.AddOrUpdate(jobName, GetHangfireJob(job), job.CronExpression);
        _logger.LogInformation("Scheduled job {JobId} with schedule {Schedule}", jobName, job.CronExpression);
    }

    #endregion Public Methods

    #region Private Methods

    private Hangfire.Common.Job GetHangfireJob(JobWorker recurringJob)
    {
        var jobType = recurringJob.GetType();
        var jobMethod = jobType.GetMethod(nameof(JobWorker.DoWork));

        return new Hangfire.Common.Job(jobMethod, jobType);
    }

    #endregion Private Methods
}