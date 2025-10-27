using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra;
using ArchlensTests.Utils;
using System.Net;

namespace ArchlensTests.Infra;

public sealed class GitSnapShotManagerTests
{
    private static Options MakeOptions(string gitUrl, string root = "/tmp/does-not-matter")
        => new(
            ProjectRoot: root,
            ProjectName: "Repo",
            Language: Language.CSharp,
            SnapshotManager: SnapshotManager.Git,
            Format: RenderFormat.Json,
            Exclusions: Array.Empty<string>(),
            FileExtensions: new[] { ".cs" },
            SnapshotDir: ".archlens",
            SnapshotFile: "snapshot.json",
            GitUrl: gitUrl
        );

    private static string BuildJson(string name, DateTime time)
        => $$"""
            {
              "name":"{{name}}",
              "lastWriteTime":"{{time}}",
              "dependencies":[],
              "children":[]
            }
            """;

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

        var graphName = "Graph";
        handler.When(mainUrl, HttpStatusCode.OK, BuildJson(graphName, DateTime.UtcNow));
        handler.When(masterUrl, HttpStatusCode.NotFound);

        var opts = MakeOptions("https://github.com/owner/repo");

        var graph = await manager.GetLastSavedDependencyGraphAsync(opts, default);

        Assert.Equal(graphName, graph.Name);
    }

    [Fact]
    public async Task Falls_Back_To_Master_When_Main_Missing()
    {
        var handler = new TestHttpHandler();
        var manager = new GitSnaphotManager(".archlens", "snapshot.json", handler);

        var mainUrl = "https://raw.githubusercontent.com/owner/repo/main/.archlens/snapshot.json";
        var masterUrl = "https://raw.githubusercontent.com/owner/repo/master/.archlens/snapshot.json";

        var graphName = "Graph";
        handler.When(mainUrl, HttpStatusCode.NotFound);
        handler.When(masterUrl, HttpStatusCode.OK, BuildJson(graphName, DateTime.UtcNow));

        var opts = MakeOptions("https://github.com/owner/repo");

        var graph = await manager.GetLastSavedDependencyGraphAsync(opts, default);

        Assert.Equal(graphName, graph.Name);
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
