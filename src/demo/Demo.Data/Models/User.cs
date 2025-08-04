namespace Demo.Data.Models;

public class User
{
    #region Public Properties

    public required string EmailAddress { get; set; }

    public ErrorType Error { get; set; }

    #endregion Public Properties
}