using ValidationTracer.Common.Jobs;

namespace ValidationTracer.ApiService.Jobs;

public class ExternalApiCollector : IRecurringJob
{
    #region Public Properties

    public string CronExpression => "0 * * * *";

    #endregion Public Properties

    #region Public Methods

    public Task DoWork(CancellationToken cancellationToken)
    {
        throw new NotImplementedException();
    }

    #endregion Public Methods
}