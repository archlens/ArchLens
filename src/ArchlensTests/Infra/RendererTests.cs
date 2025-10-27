using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using Archlens.Infra;

namespace ArchlensTests.Infra;

public sealed class RendererTests : IDisposable
{
    private readonly string _root;
    private readonly DependencyGraph _graph;

    public RendererTests()
    {
        _root = Path.Combine(Path.GetTempPath(), "archlens-tests", Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(_root);

        var leafA = new DependencyGraphLeaf() { Name = "leafA" };
        var leafB = new DependencyGraphLeaf() { Name = "leafB" };
        leafB.AddDependency("leafA");

        var graph = new DependencyGraphNode() { Name = "root" };
        var nodeA = new DependencyGraphNode() { Name = "nodeA" };
        var nodeB = new DependencyGraphNode() { Name = "nodeB" };
        leafA.AddDependency("leafB");

        nodeA.AddChild(leafA);
        nodeB.AddChild(leafB);
        graph.AddChildren([nodeA, nodeB]);

        _graph = graph;
    }

    public void Dispose()
    {
        try { Directory.Delete(_root, recursive: true); } catch { /* ignore */ }
    }

    private static Options MakeOptions(
        string projectRoot,
        IReadOnlyList<string>? exclusions = null,
        IReadOnlyList<string>? extensions = null
    )
    {
        return new Options(
            ProjectRoot: projectRoot,
            ProjectName: "TestProject",
            Language: default,
            SnapshotManager: default,
            Format: default,
            Exclusions: exclusions ?? [],
            FileExtensions: extensions ?? [".cs"]
        );
    }


    [Fact]
    public void JsonRendererRendersCorrectly()
    {
        JsonRenderer renderer = new();

        string result = renderer.RenderGraph(_graph, MakeOptions(_root));

        Assert.NotEmpty(result);
        Assert.StartsWith("{", result);
        Assert.Contains("\"title\":", result);
        Assert.Contains("\"packages\": [", result);
        Assert.Contains("\"edges\": [", result);
        Assert.Contains("\"relations\": [", result);
        Assert.EndsWith("}", result);
    }

    [Fact]
    public void PlantUMLRendererRendersCorrectly()
    {
        PlantUMLRenderer renderer = new();

        string result = renderer.RenderGraph(_graph, MakeOptions(_root));

        Assert.NotEmpty(result);
        Assert.StartsWith("@startuml", result);
        Assert.Contains("title TestProject", result);
        Assert.Contains("package \"nodeA\" as nodeA {", result);
        Assert.Contains("\"leafB\"-->leafA", result);
        Assert.EndsWith("@enduml", result);
    }

}
