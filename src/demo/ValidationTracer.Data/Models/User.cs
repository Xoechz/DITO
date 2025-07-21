namespace ValidationTracer.Data.Models;

public class User
{
    #region Public Properties

    public string? CostCenterCode { get; set; }

    public required string EmailAddress { get; set; }

    public string? ExternalApiProperty { get; set; }

    public string? ExternalDbProperty { get; set; }

    public string? InternalProperty { get; set; }

    #endregion Public Properties
}