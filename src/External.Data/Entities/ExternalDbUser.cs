using System.ComponentModel.DataAnnotations;

namespace External.Data.Entities;

public class ExternalDbUser
{
    [Key, MaxLength(50)]
    public required string EmailAddress { get; set; }

    [MaxLength(10)]
    public string? ExternalDb Property { get; set; }

    [MaxLength(4)]
    public string? CostCenterCode { get; set; }
}