using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Application;

public class DependencyGraphBuilder(IDependencyParser _dependencyParser, Options _options)
{
    public async Task<DependencyGraph> GetGraphAsync(IReadOnlyDictionary<string, IEnumerable<string>> changedModules, CancellationToken ct = default)
    {
        var graph = await BuildGraphAsync(changedModules, ct);
        return graph;
    }


    private async Task<DependencyGraphNode> BuildGraphAsync(IReadOnlyDictionary<string, IEnumerable<string>> changedModules, CancellationToken ct = default)
    {
        Dictionary<string, DependencyGraphNode> nodes = [];
        List<string> children = [];

        foreach (var module in changedModules.Keys)
        {
            DependencyGraphNode node = new() { Name = module };
            nodes.Add(module, node);
        }

        foreach (var module in changedModules.Keys)
        {
            foreach (string content in changedModules[module])
            {
                DependencyGraph child;

                var contentPath = Path.Combine(module, content);

                var relativePath = Path.GetRelativePath(_options.ProjectRoot, contentPath);
                var nameSpace = relativePath.Replace("\\", ".").Trim('.');

                var attr = File.GetAttributes(contentPath);
                if ((attr & FileAttributes.Directory) == FileAttributes.Directory)
                {
                    if (nodes.TryGetValue(content, out var existing))
                    {
                        child = existing;
                        children.Add(content);
                    }
                    else
                    {
                        var name = content.Split("\\").Last();
                        child = new DependencyGraphNode { Name = name, NameSpace = nameSpace };
                    }
                }
                else
                {
                    var deps = await _dependencyParser.ParseFileDependencies(contentPath, ct);
                    child = new DependencyGraphLeaf{ Name = content.Split("\\").Last(), NameSpace = nameSpace };
                    child.AddDependencyRange(deps);
                }
                nodes[module].AddChild(child);
            }
        }
        List<DependencyGraphNode> res = [];
        foreach(var node in nodes.Values)
        {
            if (!children.Contains(node.Name)) {
                res.Add(node);
            }
        }
        return res.FirstOrDefault();
    }
}
