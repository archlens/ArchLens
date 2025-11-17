
using Archlens.Application;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Records;
using ArchlensTests.Utils;

namespace ArchlensTests.Application;

public sealed class DependencyGraphBuilderTests : IDisposable
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


    [Fact]
    public async Task Builds_Tree_And_Aggregates_As_Expected()
    {
        Directory.CreateDirectory(Path.Combine(_fs.Root, "Domain", "Factories"));
        Directory.CreateDirectory(Path.Combine(_fs.Root, "Domain", "Interfaces"));
        Directory.CreateDirectory(Path.Combine(_fs.Root, "Domain", "Models", "Enums"));
        Directory.CreateDirectory(Path.Combine(_fs.Root, "Domain", "Models", "Records"));
        Directory.CreateDirectory(Path.Combine(_fs.Root, "Domain", "Utils"));

        var depFactory = _fs.File("Domain/Factories/DependencyParserFactory.cs", "/* */");
        var rendFactory = _fs.File("Domain/Factories/RendererFactory.cs", "/* */");
        var options = _fs.File("Domain/Models/Records/Options.cs", "/* */");
        var depGraph = _fs.File("Domain/Models/DependencyGraph.cs", "/* */");

        var domainDir = Path.Combine(_fs.Root, "Domain");
        var factoriesDir = Path.Combine(domainDir, "Factories");
        var modelsDir = Path.Combine(domainDir, "Models");
        var recordsDir = Path.Combine(modelsDir, "Records");

        var changedModules = new Dictionary<string, IEnumerable<string>>(StringComparer.OrdinalIgnoreCase)
        {
            [_fs.Root] = [Path.Combine(_fs.Root, "Domain")],
            [domainDir] = [
                Path.Combine(domainDir, "Factories"),
                Path.Combine(domainDir, "Interfaces"),
                Path.Combine(domainDir, "Models"),
                Path.Combine(domainDir, "Utils")
            ],
            [factoriesDir] = [depFactory, rendFactory],
            [modelsDir] = [Path.Combine(modelsDir, "Enums"), Path.Combine(modelsDir, "Records"), depGraph],
            [recordsDir] = [options]
        };

        var parseMap = new Dictionary<string, IReadOnlyList<string>>(StringComparer.OrdinalIgnoreCase)
        {
            [depFactory] = ["Domain.Interfaces", "Domain.Models.Enums", "Domain.Models.Records", "Infra"],
            [rendFactory] = ["Domain.Interfaces", "Domain.Models.Enums", "Infra"],
            [options] = ["Domain.Models.Enums"],
            [depGraph] = ["Domain.Utils"]
        };

        IDependencyParser parser = new TestDependencyParser(parseMap);

        var builder = new DependencyGraphBuilder(parser, MakeOptions());
        var graph = await builder.GetGraphAsync(changedModules);

        var root = graph;
        var domain = root.GetChildren().Single(c => c.Name == "Domain");
        var factories = domain.GetChildren().Single(c => c.Name == "Factories");
        var models = domain.GetChildren().Single(c => c.Name == "Models");

        Assert.True(factories.GetDependencies().ContainsKey("Infra"));
        Assert.True(factories.GetDependencies().ContainsKey("Domain.Models.Records"));
        Assert.True(factories.GetDependencies().ContainsKey("Domain.Models.Enums"));
        Assert.True(factories.GetDependencies().ContainsKey("Domain.Interfaces"));

        Assert.True(models.GetDependencies().ContainsKey("Domain.Utils"));
        Assert.False(models.GetDependencies().ContainsKey("Domain.Models.Enums"));

        var domainDeps = domain.GetDependencies().Keys;
        Assert.Contains("Infra", domainDeps);
        Assert.DoesNotContain("Domain.Interfaces", domainDeps);
        Assert.DoesNotContain("Domain.Models", domainDeps);
        Assert.DoesNotContain("Domain.Models.Enums", domainDeps);
        Assert.DoesNotContain("Domain.Utils", domainDeps);
    }
}
