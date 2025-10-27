using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Application;

public class DependencyGraphBuilder(IDependencyParser _dependencyParser, Options _options)
{
    public async Task<DependencyGraph> GetGraphAsync(string root, IReadOnlyDictionary<string, IEnumerable<string>> changedModules, CancellationToken ct = default)
    {
        DependencyGraphNode graph = new() { Name = $"{root}" };

        await BuildGraphAsync(graph, changedModules, ct);

        return graph;
    }

    private async Task BuildGraphAsync(DependencyGraphNode root, IReadOnlyDictionary<string, IEnumerable<string>> changedModules, CancellationToken ct = default)
    {
        List<DependencyGraphNode> children = [];

        foreach (var module in changedModules.Keys)
        {
            DependencyGraphNode node = new() { Name = module };

            foreach (string content in changedModules[module])
            {
                DependencyGraph child;
                var attr = File.GetAttributes(content);
                if ((attr & FileAttributes.Directory) == FileAttributes.Directory)
                    child = new DependencyGraphNode { Name = content };
                else
                {
                    var deps = await _dependencyParser.ParseFileDependencies(content, ct);
                    child = new() { Name = content.Split("\\").Last() };
                    child.AddDependencyRange(deps);
                }
                node.AddChild(child);
            }
            children.Add(node);
        }
        root.AddChildren(children);
    }
}
