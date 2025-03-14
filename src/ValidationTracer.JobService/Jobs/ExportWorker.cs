using CsvHelper;
using External.Data;
using Microsoft.EntityFrameworkCore;
using System.Globalization;
using ValidationTracer.Common.Jobs;
using ValidationTracer.Data.Models;

namespace ValidationTracer.JobService.Jobs;

public class ExportWorker(ILogger<ExportWorker> logger,
                          ExternalContext externalContext)
    : IRecurringJob
{
    #region Private Fields

    private const string API_URL = "https://localhost:7187/";

    private readonly ExternalContext _externalContext = externalContext;
    private readonly ILogger<ExportWorker> _logger = logger;

    #endregion Private Fields

    #region Public Properties

    public string CronExpression => "30 * * * *";

    #endregion Public Properties

    #region Public Methods

    public async Task DoWork(CancellationToken cancellationToken)
    {
        using var httpClient = new HttpClient();
        httpClient.BaseAddress = new Uri(API_URL);

        var response = await httpClient.GetAsync("User", cancellationToken);
        var users = await response.Content.ReadFromJsonAsync<IEnumerable<User>>(cancellationToken)
            ?? throw new InvalidOperationException("Failed to retrieve users from external API");

        var externalUsers = _externalContext.ExternalUsers.AsNoTracking().ToList();

        var result = new List<User>();

        foreach (var user in users)
        {
            var externalUser = externalUsers.FirstOrDefault(u => u.EmailAddress == user.EmailAddress);

            if (externalUser is null)
            {
                _logger.LogError("User {EmailAddress} not found in external database", user.EmailAddress);
                continue;
            }

            if (externalUser.ExternalDbProperty == null)
            {
                _logger.LogError("User {EmailAddress} ExternalDbProperty is null", user.EmailAddress);
                continue;
            }
            else if (externalUser.ExternalDbProperty.Length > 10)
            {
                _logger.LogError("User {EmailAddress} ExternalDbProperty too long", user.EmailAddress);
                continue;
            }

            if (user.InternalProperty == null)
            {
                _logger.LogError("User {EmailAddress} InternalProperty is null", user.EmailAddress);
                continue;
            }
            else if (user.InternalProperty.Length > 10)
            {
                _logger.LogError("User {EmailAddress} InternalProperty too long", user.EmailAddress);
                continue;
            }

            if (user.ExternalApiProperty == null)
            {
                _logger.LogError("User {EmailAddress} ExternalApiProperty is null", user.EmailAddress);
                continue;
            }
            else if (user.ExternalApiProperty.Length > 10)
            {
                _logger.LogError("User {EmailAddress} ExternalApiProperty too long", user.EmailAddress);
                continue;
            }

            _logger.LogInformation("User {EmailAddress} is valid", user.EmailAddress);

            user.ExternalDbProperty = externalUser.ExternalDbProperty;
            result.Add(user);
        }

        var fileName = $"export{DateTime.Now:yyyy_MM_dd-hh_mm_ss}.csv";
        string directory;

        if (Environment.OSVersion.Platform == PlatformID.Win32NT)
        {
            directory = "C:\\temp\\";
        }
        else
        {
            directory = "/tmp/";
        }

        if (!Directory.Exists(directory))
        {
            Directory.CreateDirectory(directory);
        }

        fileName = Path.Combine(directory, fileName);

        if (!File.Exists(fileName))
        {
            File.Create(fileName).Dispose();
        }

        await using var writer = new StreamWriter(fileName);
        await using var csv = new CsvWriter(writer, CultureInfo.InvariantCulture);

        csv.WriteRecords(result);
    }

    #endregion Public Methods
}