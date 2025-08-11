using Demo.Data.Models;
using Demo.Data.Repositories;
using Demo.Jobs.Config;
using Microsoft.Extensions.Options;

namespace Demo.JobService.Jobs;

public class JobWorker(ILogger<JobWorker> logger,
                       UserRepository userRepository,
                       IOptions<JobConfig> options)
{
    #region Private Fields

    private readonly ILogger<JobWorker> _logger = logger;
    private readonly IOptions<JobConfig> _options = options;
    private readonly UserRepository _userRepository = userRepository;

    #endregion Private Fields

    #region Public Properties

    public string CronExpression => _options.Value.CronExpression ?? "0 0 31 2 *";

    #endregion Public Properties

    #region Public Methods

    public async Task DoWork(CancellationToken cancellationToken)
    {
        using var httpClient = new HttpClient();
        var randomIndex = new Random().Next(0, _options.Value.TargetUrls.Count());

        httpClient.BaseAddress = new Uri(_options.Value.TargetUrls.ElementAt(randomIndex)
            ?? throw new InvalidOperationException("No target URL provided"));

        var response = await httpClient.GetAsync("User", cancellationToken);
        var users = await response.Content.ReadFromJsonAsync<IEnumerable<User>>(cancellationToken)
            ?? throw new InvalidOperationException("Failed to retrieve users from external API");
    }

    #endregion Public Methods
}