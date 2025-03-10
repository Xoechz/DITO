using System.ComponentModel.DataAnnotations;

namespace ValidationTracer.Data.Entities;

public class User
{
    #region Public Properties

    public virtual CostCenter? CostCenter { get; set; }

    [MaxLength(4)]
    public string? CostCenterCode { get; set; }

    [Key, MaxLength(50)]
    public required string EmailAddress { get; set; }

    [MaxLength(10)]
    public string? ExternalApiProperty { get; set; }

    [MaxLength(10)]
    public string? ExternalDbProperty { get; set; }

    [MaxLength(10)]
    public string? InternalProperty { get; set; }

    #endregion Public Properties
}