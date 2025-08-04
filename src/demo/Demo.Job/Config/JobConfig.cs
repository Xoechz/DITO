namespace Demo.Jobs.Config;

public class JobConfig
{
    #region Public Properties

    public string? ConnectionString { get; set; }

    public string? DatabaseName { get; set; }

    public IEnumerable<string> TargetUrls { get; set; } = [];

    public IDictionary<ErrorType, decimal> ErrorChances { get; set; } = new Dictionary<ErrorType, decimal>();

    public string? CronExpression { get; set; }

    #endregion Public Properties
}