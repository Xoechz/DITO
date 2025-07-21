using System.ComponentModel.DataAnnotations;

namespace ValidationTracer.Data.Entities;

public class CostCenter
{
    #region Public Properties

    [Key, MaxLength(4)]
    public required string Code { get; set; }

    #endregion Public Properties
}