using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;

namespace Archlens.Application;

public class DependencyGraphBuilder(IDependencyParser _dependencyParser)
{
    public async Task<DependencyGraph> GetGraphAsync(string root, IReadOnlyList<string> changedModules, CancellationToken ct = default)
    {
        string[] dir = Directory.GetDirectories(root);

        Node graph = new() { Name = "root", Children = [], Dependencies = [] };

        await BuildGraphAsync(graph, changedModules, ct);

        return graph;
    }

    private async Task BuildGraphAsync(Node root, IReadOnlyList<string> changedModules, CancellationToken ct = default)
    {
        List<Node> children = [];

        foreach (var module in changedModules)
        {

            Node node = new() { Name = module.Split("\\").Last(), Children = [], Dependencies = [] };

            string[] files = Directory.GetFiles(module);

            foreach (string file in files)
            {
                var usings = await _dependencyParser.ParseFileDependencies(file, ct);

                Leaf child = new() { Dependencies = usings, Name = file.Split("\\").Last() };

                node.AddChild(child);

                foreach (var dep in child.Dependencies)
                {
                    node.AddDependency(dep, child);
                }
            }

            string[] submodules = Directory.GetDirectories(module);

            await BuildGraphAsync(node, submodules, ct);

            children.Add(node);
        }

        root.AddChildren(children);
    }
}
