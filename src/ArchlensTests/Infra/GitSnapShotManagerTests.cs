using Archlens.Domain;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra;
using ArchlensTests.Utils;
using System.Net;

namespace ArchlensTests.Infra;

public sealed class GitSnapShotManagerTests : IDisposable
{
    private readonly TestFileSystem _fs = new();
    public void Dispose() => _fs.Dispose();
    private Options MakeOptions(string gitUrl) => new(
        ProjectRoot: _fs.Root,
        ProjectName: "Archlens",
        Language: Language.CSharp,
        SnapshotManager: SnapshotManager.Git,
        Format: RenderFormat.Json,
        Exclusions: [],
        FileExtensions: [".cs"],
        FullRootPath: _fs.Root,
        SnapshotDir: ".archlens",
        SnapshotFile: "snapshot.json",
        GitUrl: gitUrl
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
    public async Task GetLastSavedDependencyGraphAsync_Throws_When_GitUrl_Missing()
    {
        var handler = new TestHttpHandler();
        var manager = new GitSnaphotManager(".archlens", "snapshot.json", handler);

        var opts = MakeOptions(gitUrl: "  ");

        var ex = await Assert.ThrowsAsync<ArgumentException>(() => manager.GetLastSavedDependencyGraphAsync(opts, default));
        Assert.Contains("GitUrl must be provided", ex.Message);
    }

    [Theory]
    [InlineData("https://example.com/owner/repo")]
    [InlineData("https://github.com/owner")]
    [InlineData("notaurl")]
    public async Task GetLastSavedDependencyGraphAsync_Throws_When_GitUrl_Unparsable(string badUrl)
    {
        var handler = new TestHttpHandler();
        var manager = new GitSnaphotManager(".archlens", "snapshot.json", handler);

        var opts = MakeOptions(badUrl);

        var ex = await Assert.ThrowsAsync<ArgumentException>(() => manager.GetLastSavedDependencyGraphAsync(opts, default));
        Assert.Contains("Colud not parse GitUrl", ex.Message, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public async Task Returns_Graph_From_Main_When_Present()
    {
        var handler = new TestHttpHandler();
        var manager = new GitSnaphotManager(".archlens", "snapshot.json", handler);

        var mainUrl = "https://raw.githubusercontent.com/owner/repo/main/.archlens/snapshot.json";
        var masterUrl = "https://raw.githubusercontent.com/owner/repo/master/.archlens/snapshot.json";

        var graph = MakeGraph(_fs.Root);
        handler.When(mainUrl, HttpStatusCode.OK, DependencyGraphSerializer.Serialize(graph));
        handler.When(masterUrl, HttpStatusCode.NotFound);

        var opts = MakeOptions("https://github.com/owner/repo");

        var result = await manager.GetLastSavedDependencyGraphAsync(opts, default);

        Assert.Equal(graph.Name, result.Name);
    }

    [Fact]
    public async Task Falls_Back_To_Master_When_Main_Missing()
    {
        var handler = new TestHttpHandler();
        var manager = new GitSnaphotManager(".archlens", "snapshot.json", handler);

        var mainUrl = "https://raw.githubusercontent.com/owner/repo/main/.archlens/snapshot.json";
        var masterUrl = "https://raw.githubusercontent.com/owner/repo/master/.archlens/snapshot.json";

        var graph = MakeGraph(_fs.Root);
        handler.When(mainUrl, HttpStatusCode.NotFound);
        handler.When(masterUrl, HttpStatusCode.OK, DependencyGraphSerializer.Serialize(graph));

        var opts = MakeOptions("https://github.com/owner/repo");

        var results = await manager.GetLastSavedDependencyGraphAsync(opts, default);

        Assert.Equal(graph.Name, results.Name);
    }

    [Fact]
    public async Task Throws_When_Both_Branches_Missing()
    {
        var handler = new TestHttpHandler();
        var manager = new GitSnaphotManager(".archlens", "snapshot.json", handler);

        var mainUrl = "https://raw.githubusercontent.com/owner/repo/main/.archlens/snapshot.json";
        var masterUrl = "https://raw.githubusercontent.com/owner/repo/master/.archlens/snapshot.json";

        handler.When(mainUrl, HttpStatusCode.NotFound);
        handler.When(masterUrl, HttpStatusCode.NotFound);

        var opts = MakeOptions("https://github.com/owner/repo");

        var ex = await Assert.ThrowsAsync<Exception>(() => manager.GetLastSavedDependencyGraphAsync(opts, default));
        Assert.Contains("Unable to find main branch", ex.Message);
    }

}
