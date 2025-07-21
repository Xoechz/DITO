using ValidationTracer.Common.Jobs;
using ValidationTracer.Data.Models;
using ValidationTracer.Data.Repositories;

namespace ValidationTracer.ApiService.Jobs;

public class ExternalApiCollector(ILogger<ExternalApiCollector> logger,
                                  UserRepository userRepository,
                                  CostCenterRepository costCenterRepository)
    : IRecurringJob
{
    #region Private Fields

    private const string EXTERNAL_API_URL = "https://localhost:7064/";

    private readonly CostCenterRepository _costCenterRepository = costCenterRepository;
    private readonly ILogger<ExternalApiCollector> _logger = logger;
    private readonly UserRepository _userRepository = userRepository;

    #endregion Private Fields

    #region Public Properties

    public string CronExpression => "0 * * * *";

    #endregion Public Properties

    #region Public Methods

    public async Task DoWork(CancellationToken cancellationToken)
    {
        using var httpClient = new HttpClient();
        httpClient.BaseAddress = new Uri(EXTERNAL_API_URL);

        var response = await httpClient.GetAsync("User", cancellationToken);
        var users = await response.Content.ReadFromJsonAsync<IEnumerable<ExternalApiUser>>(cancellationToken)
            ?? throw new InvalidOperationException("Failed to retrieve users from external API");

        var existingUsers = await _userRepository.GetUsersAsync();
        var existingCostCenters = await _costCenterRepository.GetCostCentersAsync();

        foreach (var user in users)
        {
            var existingUser = existingUsers.FirstOrDefault(existingUser => existingUser.EmailAddress == user.EmailAddress);

            var costCenter = existingCostCenters.FirstOrDefault(costCenter => costCenter.Code == user.CostCenterCode);
            if (costCenter is null)
            {
                _logger.LogError("Cost center {CostCenterCode} not found for user {EmailAddress}", user.CostCenterCode, user.EmailAddress);
                continue;
            }

            if (user.EmailAddress is null)
            {
                _logger.LogError("User email address is null");
                continue;
            }
            else if (user.EmailAddress.Length > 20)
            {
                _logger.LogError("User email address {EmailAddress} is longer than 20 characters.", user.EmailAddress);
            }

            if (existingUser is null)
            {
                await _userRepository.AddUserAsync(new User
                {
                    EmailAddress = user.EmailAddress,
                    ExternalApiProperty = user.ExternalApiProperty,
                    CostCenterCode = user.CostCenterCode
                });
                _logger.LogInformation("User {EmailAddress} added", user.EmailAddress);
            }
            else
            {
                await _userRepository.UpdateUserAsync(new User
                {
                    EmailAddress = user.EmailAddress,
                    InternalProperty = existingUser.InternalProperty,
                    ExternalApiProperty = user.ExternalApiProperty,
                    ExternalDbProperty = existingUser.ExternalDbProperty,
                    CostCenterCode = user.CostCenterCode
                });
                _logger.LogInformation("User {EmailAddress} updated", user.EmailAddress);
            }
        }
    }

    #endregion Public Methods
}