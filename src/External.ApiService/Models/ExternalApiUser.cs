namespace External.ApiService.Models;

public class ExternalApiUser
{
    #region Public Properties

    public string? CostCenterCode { get; set; }
    public required string EmailAddress { get; set; }

    public string? ExternalApiProperty { get; set; }

    #endregion Public Properties
}