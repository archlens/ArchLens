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
        DependencyGraphNode graph = new() { Name = $"{_options.ProjectName}" };

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

                var contentPath = Path.Combine(module, content);

                var relativePath = Path.GetRelativePath(_options.ProjectRoot, contentPath);
                var nameSpace = relativePath.Replace("\\", ".").Trim('.');

                var attr = File.GetAttributes(contentPath);
                if ((attr & FileAttributes.Directory) == FileAttributes.Directory)
                {
                    var name = content.Split("\\").Last();
                    child = new DependencyGraphNode { Name = name, NameSpace = nameSpace };
                }
                else
                {
                    var deps = await _dependencyParser.ParseFileDependencies(contentPath, ct);
                    child = new DependencyGraphLeaf{ Name = content.Split("\\").Last(), NameSpace = nameSpace };
                    child.AddDependencyRange(deps);
                }
                node.AddChild(child);
            }

            children.Add(node);
        }
        root.AddChildren(children);
    }
}
