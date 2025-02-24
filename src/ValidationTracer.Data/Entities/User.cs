using System.ComponentModel.DataAnnotations;

namespace ValidationTracer.Data.Entities;

public class User
{
    [Key, MaxLength(50)]
    public required string EmailAddress { get; set; }

    [MaxLength(10)]
    public string? ExternalProperty { get; set; }

    [MaxLength(10)]
    public string? InternalProperty { get; set; }

    [MaxLength(4)]
    public string? CostCenterCode { get; set; }

    public virtual CostCenter? CostCenter { get; set; }
}