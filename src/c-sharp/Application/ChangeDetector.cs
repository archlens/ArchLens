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
        var projectRoot = string.IsNullOrEmpty(options.FullRootPath) ? Path.GetFullPath(options.ProjectRoot) : options.FullRootPath;

        var rules = CompileExclusions(options.Exclusions);
        var modules = EnumerateFiles(projectRoot, options.FileExtensions, rules);
        if (lastSavedGraph == null) return modules;

        var changed = new Dictionary<string, IEnumerable<string>>();
        var thread = new SemaphoreSlim(Environment.ProcessorCount - 1);
        var tasks = modules.Select(async pair =>
        {
            ct.ThrowIfCancellationRequested();
            await thread.WaitAsync(ct).ConfigureAwait(false);
            try
            {
                var relativePath = PathNormaliser.NormalisePath(projectRoot, pair.Key);
                if (relativePath.Equals("./"))
                    return;
                var inLastGraph = lastSavedGraph.ContainsPath(relativePath);
                if (!inLastGraph)
                {
                    changed.Add(pair.Key, pair.Value);
                }
                else
                {
                    var lastNodeWriteTime = lastSavedGraph.FindByPath(relativePath).LastWriteTime;

                    var currentWriteTime = DateTimeNormaliser.NormaliseUTC(File.GetLastWriteTimeUtc(pair.Key));

                    if (TrimMilliseconds(currentWriteTime) > TrimMilliseconds(lastNodeWriteTime))
                        changed.Add(pair.Key, pair.Value);
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

    // Source - https://stackoverflow.com/a
    // Posted by Dean Chalk, modified by community. See post 'Timeline' for change history
    // Retrieved 2025-11-23, License - CC BY-SA 4.0
    public static DateTime TrimMilliseconds(DateTime dt)
    {
        return new DateTime(dt.Year, dt.Month, dt.Day, dt.Hour, dt.Minute, dt.Second, 0, dt.Kind);
    }


    private static Dictionary<string, IEnumerable<string>> EnumerateFiles(string root, IReadOnlyList<string> extensions, ExclusionRule exclusions)
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

            try { 

                var subdirs = Directory.EnumerateDirectories(dir).Where(d => !IsExcluded(root, d, exclusions));

                foreach (var subdir in subdirs)
                    dirs.Push(subdir);

                result[dir] = result.TryGetValue(dir, out var contents)
                    ? contents.Concat(subdirs).ToList()
                    : [.. subdirs];
            } catch { /* ignore */ }
            
            IEnumerable<string> files = [];
            try { 
                files = Directory.EnumerateFiles(dir).Where(f => !IsExcluded(root, f, exclusions));
            } catch { /* ignore */ }

            var includedFiles = files.Where(file => extensions.Contains(Path.GetExtension(file)));
            result[dir] = result.TryGetValue(dir, out var existing)
                ? existing.Concat(includedFiles).ToList()
                : [.. includedFiles];
        }
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

        if (rules.DirPrefixes.Any(rule => (path + '/').StartsWith(rule, StringComparison.OrdinalIgnoreCase)))
            return true;


        var segments = path.Split('/', StringSplitOptions.RemoveEmptyEntries);
        foreach (var segment in segments)
        {
            foreach (var ban in rules.Segments)
            {
                if (MatchesSuffixPattern(segment, ban))
                    return true;
            }
        }

        var fileName = Path.GetFileName(path);
        foreach (var suf in rules.FileSuffixes)
            if (fileName.EndsWith(suf, StringComparison.OrdinalIgnoreCase))
                return true;

        return false;
    }

    public static bool MatchesSuffixPattern(string value, string pattern)
    {
        if (!pattern.Contains('*'))
            return string.Equals(value, pattern, StringComparison.Ordinal);

        var suffix = pattern.TrimStart('*');
        return value.EndsWith(suffix, StringComparison.Ordinal);
    }


    private static string GetRelative(string root, string path)
    {
        var rel = Path.GetRelativePath(root, path);
        return rel.Replace('\\', '/');
    }

}
