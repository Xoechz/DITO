using Bogus;

namespace Demo.Common.Fakers;

public abstract class CachedFakerBase<T> : Faker<T> where T : class
{
    #region Private Fields

    private List<T>? _cache;

    #endregion Private Fields

    #region Public Properties

    public List<T> Cache
    {
        get
        {
            _cache ??= Generate(CacheSize);
            return [.. _cache];
        }
    }

    public abstract int CacheSize { get; }

    #endregion Public Properties
}