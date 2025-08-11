using Demo.Data.Models;

namespace Demo.Jobs.Config;

public class JobConfig
{
    #region Public Properties

    public string? ConnectionString { get; set; }

    public string? CronExpression { get; set; }

    public string? DatabaseName { get; set; }

    public IDictionary<ErrorType, decimal> ErrorChances { get; set; } = new Dictionary<ErrorType, decimal>();

    public IEnumerable<string> TargetUrls { get; set; } = [];

    #endregion Public Properties
}