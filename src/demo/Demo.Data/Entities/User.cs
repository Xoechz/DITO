using System.ComponentModel.DataAnnotations;

namespace Demo.Data.Entities;

public class User
{
    #region Public Properties

    [Key, MaxLength(50)]
    public required string EmailAddress { get; set; }

    #endregion Public Properties
}