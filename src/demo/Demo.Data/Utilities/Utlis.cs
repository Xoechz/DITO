using Demo.Data.Models;

namespace Demo.Data.Utilities;

public static class Utlis
{
    #region Public Methods

    public static ErrorType GetRandomErrorType(IDictionary<ErrorType, decimal> errorChances)
    {
        var randomValue = (decimal)new Random().NextDouble();
        var cumulativeChance = 0.0m;

        foreach (var kvp in errorChances)
        {
            cumulativeChance += kvp.Value;
            if (randomValue < cumulativeChance)
            {
                return kvp.Key;
            }
        }

        return ErrorType.None; // Default case if no error type matches
    }

    #endregion Public Methods
}