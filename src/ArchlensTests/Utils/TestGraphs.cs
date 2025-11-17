using Archlens.Domain.Models;

namespace ArchlensTests.Utils;

public static class TestGraphs
{
    public static DependencyGraphNode Node(string projectRoot, string name, string path)
        => new(projectRoot) { Name = name, Path = path, LastWriteTime = DateTime.UtcNow };

    public static DependencyGraphLeaf Leaf(string projectRoot, string name, string path, params string[] deps)
    {
        var leaf = new DependencyGraphLeaf(projectRoot)
        {
            Name = name,
            Path = path,
            LastWriteTime = DateTime.UtcNow
        };
        leaf.AddDependencyRange(deps);
        return leaf;
    }

    public static IDictionary<string, int> Deps(this DependencyGraph g) => g.GetDependencies();
}