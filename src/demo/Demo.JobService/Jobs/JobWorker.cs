using Demo.Data.Models;
using Demo.Data.Repositories;
using Demo.JobService.Config;
using System.Diagnostics;

namespace Demo.JobService.Jobs;

public class JobWorker(ActivitySource activitySource,
                       UserRepository userRepository,
                       JobConfig config)
{
    #region Private Fields

    private readonly ActivitySource _activitySource = activitySource;
    private readonly JobConfig _config = config;
    private readonly UserRepository _userRepository = userRepository;

    #endregion Private Fields

    #region Public Properties

    public string CronExpression => _config.CronExpression ?? "0 0 31 2 *";

    #endregion Public Properties

    #region Public Methods

    public async Task DoWork(CancellationToken cancellationToken)
    {
        using var activity = _activitySource.StartActivity("Worker started", ActivityKind.Consumer);
        using var httpClient = new HttpClient();

        var randomIndex = new Random().Next(0, _config.TargetUrls.Count());
        var targetUrl = _config.TargetUrls.ElementAt(randomIndex)
            ?? throw new InvalidOperationException("No target URL provided");

        activity?.SetTag("TargetUrl", targetUrl);

        httpClient.BaseAddress = new Uri(targetUrl);

        var response = await httpClient.GetAsync("User", cancellationToken);
        var users = await response.Content.ReadFromJsonAsync<IEnumerable<User>>(cancellationToken)
            ?? throw new InvalidOperationException("Failed to retrieve users from external API");

        activity?.SetTag("UserCount", users.Count());

        foreach (var user in users)
        {
            await _userRepository.AddUserAsync(user);
        }
    }

    #endregion Public Methods
}