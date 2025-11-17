using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using Archlens.Infra;
using ArchlensTests.Utils;

namespace ArchlensTests.Infra;

public sealed class LocalSnapshotManagerTests : IDisposable
{
    private readonly TestFileSystem _fs = new();
    public void Dispose() => _fs.Dispose();

    private Options MakeOptions() => new(
        ProjectRoot: _fs.Root,
        ProjectName: "Archlens",
        Language: default,
        SnapshotManager: default,
        Format: default,
        Exclusions: [],
        FileExtensions: [".cs"],
        FullRootPath: _fs.Root
    );

    private static DependencyGraphNode MakeGraph(string rootPath)
    {
        var root = TestGraphs.Node(rootPath, "Archlens", "./");

        var domain = TestGraphs.Node(rootPath, "Domain", "./Domain/");
        var factory = TestGraphs.Node(rootPath, "Factories", "./Domain/Factories/");
        var models = TestGraphs.Node(rootPath, "Models", "./Domain/Models/");
        var records = TestGraphs.Node(rootPath, "Records", "./Domain/Models/Records/");
        var enums = TestGraphs.Node(rootPath, "Enums", "./Domain/Models/Enums/");
        var utils = TestGraphs.Node(rootPath, "Utils", "./Domain/Utils/");

        root.AddChild(domain);
        domain.AddChild(factory);
        domain.AddChild(models);
        domain.AddChild(utils);
        models.AddChild(records);
        models.AddChild(enums);

        factory.AddChild(TestGraphs.Leaf(rootPath, "DependencyParserFactory.cs",
            "./Domain/Factories/DependencyParserFactory.cs",
            "Domain.Interfaces", "Domain.Models.Enums", "Domain.Models.Records", "Infra"));

        factory.AddChild(TestGraphs.Leaf(rootPath, "RendererFactory.cs",
            "./Domain/Factories/RendererFactory.cs",
            "Domain.Interfaces", "Domain.Models.Enums", "Infra"));

        records.AddChild(TestGraphs.Leaf(rootPath, "Options.cs",
            "./Domain/Models/Records/Options.cs",
            "Domain.Models.Enums"));

        models.AddChild(TestGraphs.Leaf(rootPath, "DependencyGraph.cs",
            "./Domain/Models/DependencyGraph.cs",
            "Domain.Utils"));

        return root;
    }

    [Fact]
    public async Task SaveGraphAsync_CreatesDirectoryAndFile_AtConfiguredLocation()
    {
        // Arrange
        var dirName = ".archlens";
        var fileName = "snapshot.json";
        var snapshotManager = new LocalSnaphotManager(dirName, fileName);
        
        var opts = MakeOptions();

        var graph = MakeGraph(_fs.Root);

        var expectedDir = Path.Combine(_fs.Root, dirName);
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
        var opts = MakeOptions();

        var graph = MakeGraph(_fs.Root);

        // Act
        await snapshotManager.SaveGraphAsync(graph, opts);
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);

        // Assert
        Assert.Equal(graph.Name, loaded.Name);
        Assert.Equal(graph.LastWriteTime.ToString("dd-MM-yyyy HH:mm:ss"), loaded.LastWriteTime.ToString("dd-MM-yyyy HH:mm:ss"));
    }

    [Fact]
    public async Task GetLastSavedDependencyGraphAsync_ReturnsNull_WhenFileMissing()
    {
        // Arrange
        var snapshotManager = new LocalSnaphotManager(".archlens", "snapshot.json");
        var opts = MakeOptions();

        // Act
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);

        // Assert
        Assert.Null(loaded);
    }

    [Fact]
    public async Task Uses_CustomDirAndFileNames()
    {
        // Arrange
        var customDir = "_state";
        var customFile = "dep.json";

        var snapshotManager = new LocalSnaphotManager(customDir, customFile);
        var opts = MakeOptions();

        var graph = new DependencyGraphNode(_fs.Root) 
        { 
            Name = "CustomNames", 
            Path = customDir,
            LastWriteTime = DateTime.UtcNow 
        };
        var expectedPath = Path.Combine(_fs.Root, customDir, customFile);

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
        var opts = MakeOptions();

        var graph = MakeGraph(_fs.Root);
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
        var opts = MakeOptions();

        var graph = MakeGraph(_fs.Root);
        await snapshotManager.SaveGraphAsync(graph, opts);

        // Act
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);

        // Assert
        Assert.Equal(graph.Name, loaded.Name);
        Assert.Single(loaded.GetChildren());
        var domain = loaded.GetChild("./Domain");
        Assert.NotNull(domain);
        Assert.Equal(3, domain.GetChildren().Count);
    }

    [Fact]
    public async Task Load_ReturnsSubModuleDependencies_WhenPresent()
    {
        // Arrange
        var snapshotManager = new LocalSnaphotManager(".archlens", "snapshot.json");
        var opts = MakeOptions();

        var graph = MakeGraph(_fs.Root);
        await snapshotManager.SaveGraphAsync(graph, opts);

        // Act
        var loaded = await snapshotManager.GetLastSavedDependencyGraphAsync(opts);
        var domain = loaded.GetChild("./Domain");
        // Assert
        Assert.NotNull(domain);
        Assert.Empty(loaded.GetDependencies());
        Assert.Single(domain.GetDependencies());
    }
}