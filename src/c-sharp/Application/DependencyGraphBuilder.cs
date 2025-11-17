using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Application;

public class DependencyGraphBuilder(IDependencyParser _dependencyParser, Options _options)
{
    public async Task<DependencyGraph> GetGraphAsync(
        IReadOnlyDictionary<string, IEnumerable<string>> changedModules,
        CancellationToken ct = default)
    {
        var root = await BuildGraphAsync(changedModules, ct);
        DependencyAggregator.RecomputeAggregates(root);
        return root;
    }

    private static string CanonRel(string root, string path)
    {
        var abs = Path.GetFullPath(path);
        var rel = Path.GetRelativePath(root, abs);
        rel = rel.Replace(Path.AltDirectorySeparatorChar, Path.DirectorySeparatorChar).TrimEnd(Path.DirectorySeparatorChar);
        return rel.Length == 0 ? "." : rel;
    }

    private static string JoinRel(string parentRel, string child)
    {
        var p = parentRel == "." ? child : Path.Combine(parentRel, child);
        return p.Replace(Path.AltDirectorySeparatorChar, Path.DirectorySeparatorChar)
                .TrimEnd(Path.DirectorySeparatorChar);
    }

    private async Task<DependencyGraphNode> BuildGraphAsync(
        IReadOnlyDictionary<string, IEnumerable<string>> changedModules,
        CancellationToken ct)
    {
        var comparer = StringComparer.OrdinalIgnoreCase;
        var nodes = new Dictionary<string, DependencyGraphNode>(comparer);

        var rootNode = new DependencyGraphNode(_options.FullRootPath)
        {
            Name = _options.ProjectName,
            Path = _options.ProjectRoot,
            LastWriteTime = File.GetLastWriteTimeUtc(_options.FullRootPath)
        };
        nodes["."] = rootNode;

        foreach (var moduleAbs in changedModules.Keys)
        {
            var key = CanonRel(_options.FullRootPath, moduleAbs);
            var name = key == "." ? _options.ProjectName : Path.GetFileName(key);
            nodes[key] = new DependencyGraphNode(_options.FullRootPath)
            {
                Name = name,
                Path = moduleAbs,
                LastWriteTime = File.GetLastWriteTimeUtc(moduleAbs)
            };
        }

        foreach (var key in nodes.Keys.OrderBy(k => k.Count(c => c == Path.DirectorySeparatorChar)))
        {
            if (key == ".") continue;
            var parentKey = Path.GetDirectoryName(key);
            if (string.IsNullOrEmpty(parentKey)) parentKey = ".";
            nodes[parentKey].AddChild(nodes[key]);
        }

        foreach (var (moduleAbs, contents) in changedModules)
        {
            var moduleKey = CanonRel(_options.FullRootPath, moduleAbs);
            var moduleNode = nodes[moduleKey];

            foreach (var absPath in contents)
            {
                var childKey = JoinRel(moduleKey, absPath);

                if (nodes.TryGetValue(childKey, out var childDir))
                {
                    moduleNode.AddChild(childDir);
                    continue;
                }

                if (!Path.HasExtension(absPath)) continue;

                var deps = await _dependencyParser.ParseFileDependencies(absPath, ct).ConfigureAwait(false);
                var leaf = new DependencyGraphLeaf(_options.FullRootPath)
                {
                    Name = Path.GetFileName(childKey),
                    Path = $"{absPath}",
                    LastWriteTime = File.GetLastWriteTimeUtc(absPath)
                };
                leaf.AddDependencyRange(deps);
                moduleNode.AddChild(leaf);
            }
        }

        return nodes["."]; // project root
    }
}

