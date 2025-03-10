using System.ComponentModel.DataAnnotations;

namespace External.Data.Entities;

public class ExternalDbUser
{
    #region Public Properties

    [Key, MaxLength(50)]
    public required string EmailAddress { get; set; }

    [MaxLength(10)]
    public string? ExternalDbProperty { get; set; }

    #endregion Public Properties
}