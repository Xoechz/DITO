namespace Demo.AppHost;

internal class DynamicService(int index)
{
    #region Internal Properties

    internal IResourceBuilder<SqlServerDatabaseResource>? DataBase { get; set; }

    internal int Index { get; set; } = index;

    internal IResourceBuilder<ProjectResource>? MigrationService { get; set; }

    internal string Name => "Service" + Index;

    #endregion Internal Properties
}