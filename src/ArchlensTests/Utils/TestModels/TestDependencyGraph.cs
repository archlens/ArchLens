using Archlens.Domain.Models;

namespace ArchlensTests.Utils.TestModels;

public sealed class TestDependencyGraph(string projectRootPath) : DependencyGraph(projectRootPath)
{
    private readonly Dictionary<string, TestNode> _nodes =
        new(StringComparer.OrdinalIgnoreCase);

    public void SetFile(string relativePath, DateTime lastWriteUtc)
    {
        var pair = Normalize(relativePath);
        var nameIndx = pair.LastIndexOf('/');
        var key = pair[..nameIndx];
        _nodes[key] = new TestNode("") // TODO: fix in other PR
        {
            Name = key,
            Path = key,
            LastWriteTime = DateTime.SpecifyKind(lastWriteUtc, DateTimeKind.Utc)
        };
    }


    public override DependencyGraph GetChild(string relativePath)
    {
        var key = Normalize(relativePath);
        if (!_nodes.TryGetValue(key, out var node))
            throw new KeyNotFoundException($"No node for '{relativePath}'");
        return node;
    }

    private sealed class TestNode(string projectRootPath) : DependencyGraph(projectRootPath) { }

    private static string Normalize(string p) => p.Replace('\\', '/');
}