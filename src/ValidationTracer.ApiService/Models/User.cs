namespace ValidationTracer.ApiService.Models;

public class User
{
    public required string EmailAddress { get; set; }

    public string? ExternalDbProperty { get; set; }
    public string? ExternalApiProperty { get; set; }

    public string? InternalProperty { get; set; }

    public string? CostCenterCode { get; set; }
}