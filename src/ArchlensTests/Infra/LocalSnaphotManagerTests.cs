using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using Archlens.Infra;

namespace ArchlensTests.Infra;

public sealed class LocalSnapshotManagerTests : IDisposable
{
    private readonly string _tempRoot;

    public LocalSnapshotManagerTests()
    {
        _tempRoot = Path.Combine(Path.GetTempPath(), "archlens-tests", Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(_tempRoot);
    }

    public void Dispose()
    {
        try { Directory.Delete(_tempRoot, recursive: true); } catch { /* ignore */ }
    }

    private static Options MakeOptions(string projectRoot) =>
        new(
            ProjectRoot: projectRoot,
            ProjectName: "TestProject",
            Language: default,
            SnapshotManager: default,
            Format: default,
            Exclusions: [],
            FileExtensions: [".cs"]
        );

    private static DependencyGraphNode MakeGraph()
    {
        var leafA = new DependencyGraphLeaf(){ Name = "ModuleA.FileA", LastWriteTime = DateTime.UtcNow.AddHours(-1) };
        var leafB = new DependencyGraphLeaf(){ Name = "ModuleB.FileB", LastWriteTime = DateTime.UtcNow };

        var nodeA = new DependencyGraphNode(){ Name = "ModuleA", LastWriteTime = DateTime.UtcNow.AddHours(-1) };
        nodeA.AddChild(leafA);
        
        leafB.AddDependency(leafA.Name);

        var nodeB = new DependencyGraphNode(){ Name = "ModuleB", LastWriteTime = DateTime.UtcNow };
        nodeB.AddChild(leafB);

        var graph = new DependencyGraphNode() { Name = "Root", LastWriteTime = DateTime.UtcNow.AddHours(-1) };
        graph.AddChildren([ nodeA, nodeB ]);

        return graph;
    }

    [Fact]
    public async Task SaveGraphAsync_CreatesDirectoryAndFile_AtConfiguredLocation()
    {
        // Arrange
        var dirName = ".archlens";
        var fileName = "snapshot.json";
        var snapshotManager = new LocalSnaphotManager(dirName, fileName);
        
        var opts = MakeOptions(_tempRoot);

        var graph = MakeGraph();

        var expectedDir = Path.Combine(_tempRoot, dirName);
        var expectedFile = Path.Combine(expectedDir, fileName);

        // Act
        await snapshotManager.SaveGraphAsync(graph, opts);

        // Assert
        Assert.True(Directory.Exists(expectedDir));
        Assert.True(File.Exists(expectedFile));

        var json = await File.ReadAllTextAsync(expectedFile);
        Assert.Contains(graph.Name, json);
    }

    [Fact]
    public async Task SaveThenLoad_Get_Name_And_LastWriteTime()
    {
        // Arrange
        var snapshotManager = new LocalSnaphotManager(".archlens", "snapshot.json");
        var opts = MakeOptions(_tempRoot);

        var graph = MakeGraph();

        // Act
        await snapshotManager.SaveGraphAsync(graph, opts);
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);

        // Assert
        Assert.Equal(graph.Name, loaded.Name);
        Assert.Equal(graph.LastWriteTime.ToString("dd-MM-yyyy HH:mm:ss"), loaded.LastWriteTime.ToString("dd-MM-yyyy HH:mm:ss"));
    }

    [Fact]
    public async Task GetLastSavedDependencyGraphAsync_ReturnsDefault_WhenFileMissing()
    {
        // Arrange
        var snapshotManager = new LocalSnaphotManager(".archlens", "snapshot.json");
        var opts = MakeOptions(_tempRoot);

        // Act
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);

        // Assert
        Assert.StartsWith("Snapshot@", loaded.Name, StringComparison.Ordinal);
    }

    [Fact]
    public async Task Uses_CustomDirAndFileNames()
    {
        // Arrange
        var customDir = "_state";
        var customFile = "dep.json";
        var snapshotManager = new LocalSnaphotManager(customDir, customFile);
        var opts = MakeOptions(_tempRoot);

        var graph = new DependencyGraphNode { Name = "CustomNames", LastWriteTime = DateTime.UtcNow };
        var expectedPath = Path.Combine(_tempRoot, customDir, customFile);

        // Act
        await snapshotManager.SaveGraphAsync(graph, opts);

        // Assert
        Assert.True(File.Exists(expectedPath));
    }

    [Fact]
    public async Task Load_ReturnsGraph_WhenFilePresent()
    {
        // Arrange
        var snapshotManager = new LocalSnaphotManager(".archlens", "snapshot.json");
        var opts = MakeOptions(_tempRoot);

        var graph = MakeGraph();
        await snapshotManager.SaveGraphAsync(graph, opts);

        // Act
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);

        // Assert
        Assert.Equal(graph.Name, loaded.Name);
    }

    [Fact]
    public async Task Load_ReturnsMultiLevelGraph_WhenPresent()
    {
        // Arrange
        var snapshotManager = new LocalSnaphotManager(".archlens", "snapshot.json");
        var opts = MakeOptions(_tempRoot);

        var graph = MakeGraph();
        await snapshotManager.SaveGraphAsync(graph, opts);

        // Act
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);

        // Assert
        Assert.Equal(graph.Name, loaded.Name);
        Assert.Equal(2, loaded.GetChildren().Count);
    }

    [Fact]
    public async Task Load_ReturnsSubModuleDependencies_WhenPresent()
    {
        // Arrange
        var snapshotManager = new LocalSnaphotManager(".archlens", "snapshot.json");
        var opts = MakeOptions(_tempRoot);

        var graph = MakeGraph();
        await snapshotManager.SaveGraphAsync(graph, opts);

        // Act
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);
        var moduleB = loaded.GetChild("ModuleB");

        // Assert
        Assert.NotNull(moduleB);
        Assert.Empty(loaded.GetDependencies());
        Assert.Single(moduleB.GetDependencies());
    }
}
