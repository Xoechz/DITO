using System.ComponentModel.DataAnnotations;

namespace External.Data.Entities;

public class ExternalUser
{
    [Key, MaxLength(50)]
    public required string EmailAddress { get; set; }

    [MaxLength(10)]
    public string? ExternalProperty { get; set; }

    [MaxLength(4)]
    public string? CostCenterCode { get; set; }
}