using Archlens.Application;
using Archlens.Domain.Models;
using ArchlensTests.Utils;

namespace ArchlensTests.Application;

public sealed class DependencyAggregatorTests
{
    private static DependencyGraphNode MakeDomainTree()
    {
        var root = TestGraphs.Node(".", "Archlens", "./");

        var domain = TestGraphs.Node(".", "Domain", "./Domain/");
        var factory = TestGraphs.Node(".", "Factories", "./Domain/Factories/");
        var models = TestGraphs.Node(".", "Models", "./Domain/Models/");
        var records = TestGraphs.Node(".", "Records", "./Domain/Models/Records/");
        var enums = TestGraphs.Node(".", "Enums", "./Domain/Models/Enums/");
        var utils = TestGraphs.Node(".", "Utils", "./Domain/Utils/");

        root.AddChild(domain);
        domain.AddChild(factory);
        domain.AddChild(models);
        domain.AddChild(utils);
        models.AddChild(records);
        models.AddChild(enums);

        factory.AddChild(TestGraphs.Leaf(".", "DependencyParserFactory.cs",
            "./Domain/Factories/DependencyParserFactory.cs",
            "Domain.Interfaces", "Domain.Models.Enums", "Domain.Models.Records", "Infra"));

        factory.AddChild(TestGraphs.Leaf(".", "RendererFactory.cs",
            "./Domain/Factories/RendererFactory.cs",
            "Domain.Interfaces", "Domain.Models.Enums", "Infra"));

        records.AddChild(TestGraphs.Leaf(".", "Options.cs",
            "./Domain/Models/Records/Options.cs",
            "Domain.Models.Enums"));

        models.AddChild(TestGraphs.Leaf(".", "DependencyGraph.cs",
            "./Domain/Models/DependencyGraph.cs",
            "Domain.Utils"));

        return root;
    }

    [Fact]
    public void Aggregation_Drops_Internal_Subtree_Relations_And_Preserves_External()
    {
        var root = MakeDomainTree();
        DependencyAggregator.RecomputeAggregates(root);

        var domain = root.GetChildren().Single(c => c.Name == "Domain");
        var factory = domain.GetChildren().Single(c => c.Name == "Factories");
        var models = domain.GetChildren().Single(c => c.Name == "Models");

        Assert.True(factory.GetDependencies().ContainsKey("Domain.Interfaces"));
        Assert.True(factory.GetDependencies().ContainsKey("Domain.Models.Enums"));
        Assert.True(factory.GetDependencies().ContainsKey("Domain.Models.Records"));
        Assert.True(factory.GetDependencies().ContainsKey("Infra"));

        Assert.True(models.GetDependencies().ContainsKey("Domain.Utils"));
        Assert.False(models.GetDependencies().ContainsKey("Domain.Models.Enums"));

        var domainDeps = domain.GetDependencies().Keys;
        Assert.DoesNotContain("Domain.Models.Enums", domainDeps);
        Assert.DoesNotContain("Domain.Models.Records", domainDeps);
        Assert.DoesNotContain("Domain.Utils", domainDeps);
        Assert.DoesNotContain("Domain.Interfaces", domainDeps);
        Assert.Contains("Infra", domainDeps);
    }

    [Fact]
    public void Root_Shows_All_External_Relations()
    {
        var root = MakeDomainTree();
        DependencyAggregator.RecomputeAggregates(root);

        var rootDeps = root.GetDependencies().Keys;
        Assert.Contains("Infra", rootDeps);
    }
}
