using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using Archlens.Domain.Utils;
using System;
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

        foreach (var modulePath in changedModules.Keys)
        {
            var name = modulePath.Equals(_options.FullRootPath) ? _options.ProjectName : Path.GetFileName(modulePath);
            nodes[modulePath] = new DependencyGraphNode(_options.FullRootPath)
            { 
                Name = name,
                Path = modulePath,
                LastWriteTime = File.GetLastWriteTimeUtc(modulePath) 
            };
        }

        foreach (var pair in changedModules)
        {
            var module = pair.Key;
            var contents = pair.Value;

            var parent = nodes[module];

            foreach (var content in contents)
            {
                var contentPath = Path.Combine(module, content);
                var nameSpace   = GetNameSpace(contentPath);

                DependencyGraph child;

                bool isDir;
                try
                {
                    isDir = (File.GetAttributes(contentPath) & FileAttributes.Directory) != 0;
                }
                catch { continue; }
                if (isDir)
                {
                    if (nodes.TryGetValue(content, out var existing))
                    {
                        child = existing;
                        children.Add(content);
                    }
                    else
                    {
                        var name = Path.GetFileName(content);
                        child = new DependencyGraphNode(_options.FullRootPath)
                        { 
                            Name = name, 
                            Path = contentPath,
                            LastWriteTime = File.GetLastWriteTimeUtc(module) 
                        };
                    }
                }
                else
                {
                    var name = Path.GetFileName(content);
                    var deps = await _dependencyParser.ParseFileDependencies(contentPath, ct).ConfigureAwait(false);
                    var leaf = new DependencyGraphLeaf(_options.FullRootPath)
                    { 
                        Name = name, 
                        Path = contentPath,
                        LastWriteTime = File.GetLastWriteTimeUtc(contentPath)
                    };
                    leaf.AddDependencyRange(deps);
                    child = leaf;
                }
                parent.AddChild(child);
            }
        }
        var rootKey = nodes.Keys
        .Except(children, StringComparer.OrdinalIgnoreCase)
        .OrderBy(k => !k.Equals(_options.FullRootPath, StringComparison.OrdinalIgnoreCase))
        .FirstOrDefault();

        return rootKey is null ? null : nodes[rootKey];
    }

    private string GetNameSpace(string fullPath)
    {
        var relPath = Path.GetRelativePath(_options.FullRootPath, fullPath);
        var nameSpace = relPath.Replace(Path.DirectorySeparatorChar, '.')
                               .Replace(Path.AltDirectorySeparatorChar, '.')
                               .Trim('.');
        return nameSpace;
    }
}
