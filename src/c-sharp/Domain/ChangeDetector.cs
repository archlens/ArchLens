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

    public static async Task<IReadOnlyList<string>> GetChangedProjectFilesAsync(
        Options options,
        DependencyGraph lastSavedGraph,
        CancellationToken ct = default)
    {
        var projectRoot = Path.GetFullPath(options.ProjectRoot);

        var candidates = EnumerateFiles(projectRoot, options.FileExtensions);

        var files = candidates.ToArray();

        if (options.Exclusions != null && options.Exclusions.Count > 0)
        {
            var exclusions = CompileExclusions(options.Exclusions);
            files = [.. candidates.Where(path => !IsExcluded(projectRoot, path, exclusions))];
        }

        var changed = new List<string>();
        var thread = new SemaphoreSlim(Environment.ProcessorCount - 1);

        var tasks = files.Select(async file =>
        {
            ct.ThrowIfCancellationRequested();
            await thread.WaitAsync(ct).ConfigureAwait(false);
            try
            {
                var relativePath = GetRelative(projectRoot, file);
                var inLastGraph = lastSavedGraph.Packages().Contains(relativePath);
                if (!inLastGraph)
                {
                    changed.Add(relativePath);
                }
                else
                {
                    var lastNodeWriteTime = lastSavedGraph.GetChild(relativePath).LastWriteTime;
                    var currentWriteTime = File.GetLastWriteTimeUtc(file);

                    if (currentWriteTime > lastNodeWriteTime)
                        changed.Add(relativePath);
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

    private static IEnumerable<string> EnumerateFiles(string root, IReadOnlyList<string> extensions)
    {
        var dirs = new Stack<string>();
        dirs.Push(root);

        while (dirs.Count > 0)
        {
            var dir = dirs.Pop();

            IEnumerable<string> subdirs = [];
            try { 
                subdirs = Directory.EnumerateDirectories(dir);
            } catch { /* ignore */ }
            
            foreach (var subdir in subdirs) 
                dirs.Push(subdir);

            IEnumerable<string> files = [];
            try { 
                files = Directory.EnumerateFiles(dir);
            } catch { /* ignore */ }

            foreach (var file in files)
            {
                if (extensions.Contains(Path.GetExtension(file)))
                    yield return file;
            }
        }
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

    private static bool IsExcluded(string projectRoot, string absolutePath, ExclusionRule rules)
    {
        var relativePath = GetRelative(projectRoot, absolutePath);

        var fileName = Path.GetFileName(relativePath);

        foreach (var prefix in rules.DirPrefixes)
            if (relativePath.StartsWith(prefix, StringComparison.OrdinalIgnoreCase))
                return true;

        var segments = relativePath.Split('/', StringSplitOptions.RemoveEmptyEntries);
        foreach (var segment in segments)
        {
            foreach (var ban in rules.Segments)
            {
                if (segment.Equals(ban, StringComparison.OrdinalIgnoreCase))
                    return true;
            }
        }

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
