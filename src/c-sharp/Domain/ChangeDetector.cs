using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Domain;

public sealed class ChangeDetector
{
    private sealed record ExclusionRule(
        string[] DirPrefixes,      // strings that end with a '/', to indicate that it is a dir, like: "src/legacy/", "Tests/"
        string[] Segments,         // folders that are often mid-path, like: "bin", "obj", ".git"
        string[] FileSuffixes      // specific file postfixes, like: "*.dev.cs", ".g.cs"
    );

    public static async Task<IReadOnlyDictionary<string, IEnumerable<string>>> GetChangedProjectFilesAsync(
        Options options,
        DependencyGraph lastSavedGraph,
        CancellationToken ct = default)
    {
        var projectRoot = String.IsNullOrEmpty(options.FullRootPath) ? Path.GetFullPath(options.ProjectRoot) : options.FullRootPath;

        var candidates = EnumerateFiles(projectRoot, options.FileExtensions);

        Dictionary<string, IEnumerable<string>> modules = [];
        if (options.Exclusions != null && options.Exclusions.Count > 0)
        {
            var rules = CompileExclusions(options.Exclusions);
            foreach(var module in candidates.Keys)
            {
                if (module == projectRoot)
                    continue;
                var isExcludedModule = IsExcluded(projectRoot, module, rules);
                if (!isExcludedModule)
                {
                    var contents = candidates[module].Where(content => !IsExcluded(projectRoot, content, rules));
                    modules.Add(module, contents);
                }
            }
        }
        else
        {
            modules = candidates;
        }

        var changed = new Dictionary<string, IEnumerable<string>>();
        var thread = new SemaphoreSlim(Environment.ProcessorCount - 1);
        var tasks = modules.Select(async pair =>
        {
            ct.ThrowIfCancellationRequested();
            await thread.WaitAsync(ct).ConfigureAwait(false);
            try
            {
                var relativePath = GetRelative(projectRoot, pair.Key);
                var inLastGraph = lastSavedGraph.Packages().Contains(relativePath);
                if (!inLastGraph)
                {
                    changed.Add(relativePath, pair.Value);
                }
                else
                {
                    var lastNodeWriteTime = lastSavedGraph.GetChild(relativePath).LastWriteTime;

                    var currentWriteTime = File.GetLastWriteTimeUtc(pair.Key);

                    if (currentWriteTime > lastNodeWriteTime)
                        changed.Add(relativePath, pair.Value);
                }
            }
            finally
            {
                thread.Release();
            }
        });

        await Task.WhenAll(tasks).ConfigureAwait(false);
        return changed;
    }

    private static Dictionary<string, IEnumerable<string>> EnumerateFiles(string root, IReadOnlyList<string> extensions)
    {
        var dirs = new Stack<string>();
        dirs.Push(root);

        var result = new Dictionary<string, IEnumerable<string>>
        {
            { root, [] }
        };

        while (dirs.Count > 0)
        {
            var dir = dirs.Pop();

            IEnumerable<string> subdirs = [];
            try { 

                subdirs = Directory.EnumerateDirectories(dir);
            } catch { /* ignore */ }
            
            foreach (var subdir in subdirs)
                dirs.Push(subdir);
            
            result[dir] = [..subdirs];

            IEnumerable<string> files = [];
            try { 
                files = Directory.EnumerateFiles(dir);
            } catch { /* ignore */ }

            var includedFiles = files.Where(file => extensions.Contains(Path.GetExtension(file))).Select(file => GetRelative(root, file).Split('/').Last()).ToList();
            result[dir] = [.. includedFiles];
        }
        result.Remove(root);
        return result;
    }


    private static ExclusionRule CompileExclusions(IReadOnlyList<string> exclusions)
    {
        var dirPrefixes = new List<string>();
        var segments = new List<string>();
        var suffixes = new List<string>();

        foreach (var entry in exclusions)
        {
            var exclusion = (entry ?? string.Empty).Trim();
            if (exclusion.Length == 0) continue;

            if (exclusion.StartsWith("**/", StringComparison.Ordinal)) exclusion = exclusion[3..];

            var norm = exclusion.Replace('\\', '/');
            if (norm.EndsWith('.')) norm = norm[..^1];

            // relative path with trailing '/' -> dir
            if (norm.EndsWith('/'))
            {
                var p = norm;
                if (p.StartsWith("./", StringComparison.Ordinal)) p = p[2..];
                if (!p.EndsWith('/')) p += "/";
                dirPrefixes.Add(p);
                continue;
            }

            // a relative path without trailing '/' -> dir
            if (norm.Contains('/'))
            {
                var p = norm;
                if (!p.EndsWith('/')) p += "/";
                dirPrefixes.Add(p);
                continue;
            }

            // Filename wildcard like "*.dev.cs" -> suffix on filename
            if (norm.StartsWith("*.", StringComparison.Ordinal))
            {
                suffixes.Add(norm[1..]);
                continue;
            }

            segments.Add(norm.TrimStart('.'));
        }

        return new ExclusionRule(
            DirPrefixes: [.. dirPrefixes],
            Segments: [.. segments],
            FileSuffixes: [.. suffixes]
        );
    }

    private static bool IsExcluded(string projectRoot, string content, ExclusionRule rules)
    {
        var path = GetRelative(projectRoot, content);

        if (rules.DirPrefixes.Any(rule => rule.StartsWith(path, StringComparison.OrdinalIgnoreCase)))
            return true;


        var segments = path.Split('/', StringSplitOptions.RemoveEmptyEntries);
        foreach (var segment in segments)
        {
            foreach (var ban in rules.Segments)
            {
                if (segment.Equals(ban, StringComparison.OrdinalIgnoreCase))
                    return true;
            }
        }

        var fileName = Path.GetFileName(path);
        foreach (var suf in rules.FileSuffixes)
            if (fileName.EndsWith(suf, StringComparison.OrdinalIgnoreCase))
                return true;

        return false;
    }

    private static string GetRelative(string root, string path)
    {
        var rel = Path.GetRelativePath(root, path);
        return rel.Replace('\\', '/');
    }

}
