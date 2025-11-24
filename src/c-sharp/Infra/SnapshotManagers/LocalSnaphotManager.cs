using Archlens.Domain;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using System.IO;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Infra.SnapshotManagers;

public sealed class LocalSnaphotManager(string _localDirName, string _localFileName) : ISnapshotManager
{
    public async Task SaveGraphAsync(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        var root = string.IsNullOrEmpty(options.FullRootPath) ? Path.GetFullPath(options.ProjectRoot) : options.FullRootPath;

        var dir = Path.Combine(root, _localDirName);

        Directory.CreateDirectory(dir);
        var path = Path.Combine(dir, _localFileName);

        var json = DependencyGraphSerializer.Serialize(graph);

        await File.WriteAllTextAsync(path, json, ct);
    }

    public async Task<DependencyGraph> GetLastSavedDependencyGraphAsync(Options options, CancellationToken ct = default)
    {
        var root = string.IsNullOrEmpty(options.FullRootPath) ? Path.GetFullPath(options.ProjectRoot) : options.FullRootPath;
        var path = Path.Combine(root, _localDirName, _localFileName);

        if (!File.Exists(path))
        {
            return null;
        }

        var json = await File.ReadAllTextAsync(path, ct);

        var graph = DependencyGraphSerializer.Deserialize(json, root);

        return graph ?? null;
    }
}