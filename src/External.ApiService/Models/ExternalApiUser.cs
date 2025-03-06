namespace External.ApiService.Models;

public class ExternalApiUser
{
    public required string EmailAddress { get; set; }

    public string? ExternalApiProperty { get; set; }

    public string? CostCenterCode { get; set; }
}