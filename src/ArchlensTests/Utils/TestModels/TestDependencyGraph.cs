using Archlens.Domain.Models;

namespace ArchlensTests.Utils.TestModels;

public sealed class TestDependencyGraph : DependencyGraph
{
    private readonly Dictionary<string, TestNode> _nodes =
        new(StringComparer.OrdinalIgnoreCase);

    public void SetFile(string relativePath, DateTime lastWriteUtc)
    {
        var pair = Normalize(relativePath);
        var nameIndx = pair.LastIndexOf('/');
        var key = pair[..nameIndx];
        _nodes[key] = new TestNode
        {
            Name = key,
            LastWriteTime = DateTime.SpecifyKind(lastWriteUtc, DateTimeKind.Utc)
        };
    }

    public override List<string> Packages() => [.. _nodes.Keys];

    public override DependencyGraph GetChild(string relativePath)
    {
        var key = Normalize(relativePath);
        if (!_nodes.TryGetValue(key, out var node))
            throw new KeyNotFoundException($"No node for '{relativePath}'");
        return node;
    }

    private sealed class TestNode : DependencyGraph { }

    private static string Normalize(string p) => p.Replace('\\', '/');
}