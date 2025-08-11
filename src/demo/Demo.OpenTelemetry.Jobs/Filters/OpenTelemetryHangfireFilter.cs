using Hangfire.Server;
using Hangfire.States;
using Hangfire.Storage;
using System.Diagnostics;

namespace Demo.OpenTelemetry.Jobs.Filters;

/// <summary>
/// A filter that captures job failure events in Hangfire and logs them using OpenTelemetry.
/// No Attribute is needed to use this filter, it is automatically applied to all Hangfire jobs.
/// </summary>
/// <param name="activitySource"><see cref="ActivitySource"/> to create an OpenTelemetry Trace</param>
public class OpenTelemetryHangfireFilter(ActivitySource activitySource) : IApplyStateFilter, IServerFilter
{
    #region Private Fields

    private readonly Dictionary<string, Activity> _activities = [];
    private readonly ActivitySource _activitySource = activitySource;

    #endregion Private Fields

    #region Public Methods

    /// <inheritdoc/>
    public void OnPerformed(PerformedContext context)
    {
        // is handled in OnStateApplied to detect final failures
    }

    /// <inheritdoc/>
    public void OnPerforming(PerformingContext context)
    {
        // This method is called before the job is executed.
        // It is used to start a new activity for the job.
        var jobId = context.BackgroundJob.Id;

        if (_activities.TryGetValue(jobId, out var existingActivity))
        {
            _activities.Remove(jobId);
            existingActivity.SetStatus(ActivityStatusCode.Error, "Activity has not been disposed correctly.");
            existingActivity.AddEvent(new ActivityEvent("Activity has not been disposed correctly."));
            existingActivity.Dispose();
        }

        var jobName = context.GetJobParameter<string>("RecurringJobId");
        var activity = _activitySource.StartActivity("HangfireJobPerforming");

        if (activity is null)
        {
            return; // If activity creation fails, we exit early
        }

        activity.DisplayName = $"JOB {jobName} performing";
        activity.SetTag("job.name", jobName);
        activity.SetTag("job.id", context.BackgroundJob.Id);
        activity.SetTag("job.type", context.BackgroundJob.Job.Type.FullName);
        activity.SetTag("job.method", context.BackgroundJob.Job.Method.Name);

        _activities.Add(jobId, activity);
    }

    /// <inheritdoc/>
    public void OnStateApplied(ApplyStateContext context, IWriteOnlyTransaction transaction)
    {
        // This method is called when the job state is applied. If the activity is set, this means the job was executed before.
        // The activity is disposed in this method to end the trace.
        var jobId = context.BackgroundJob.Id;

        if (!_activities.TryGetValue(jobId, out var activity))
        {
            return; // If no activity exists for this job, we exit early
        }

        _activities.Remove(jobId);

        if (context.NewState is FailedState failedState)
        {
            // Add a failure trace, to make finding the error easier in Application Insights
            using var errorActivity = _activitySource.StartActivity("JobFailureAfterRetries");
            errorActivity?.SetTag("exception.message", failedState.Exception.Message);
            errorActivity?.SetTag("exception.stacktrace", failedState.Exception.StackTrace);
            errorActivity?.SetTag("retries", context.GetJobParameter<int>("RetryCount"));
            errorActivity?.SetStatus(ActivityStatusCode.Error, "Job failed after all retries");
            activity.SetStatus(ActivityStatusCode.Error, "Job failed after all retries");
            errorActivity?.AddException(failedState.Exception);
            errorActivity?.AddEvent(new ActivityEvent("JobFailureAfterRetries"));
        }

        activity.SetTag("job.state", context.NewState.Name);
        activity.SetTag("job.state.reason", context.NewState.Reason ?? "No reason provided");
        activity.Dispose();
    }

    /// <inheritdoc/>
    public void OnStateUnapplied(ApplyStateContext context, IWriteOnlyTransaction transaction)
    {
        // No action needed when state is unapplied
    }

    #endregion Public Methods
}