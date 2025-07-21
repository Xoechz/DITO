namespace ValidationTracer.Common.Jobs;

public interface IRecurringJob
{
    #region Public Properties

    string CronExpression { get; }

    #endregion Public Properties

    #region Public Methods

    Task DoWork(CancellationToken cancellationToken);

    #endregion Public Methods
}