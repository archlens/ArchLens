
using Archlens.Domain.Models;
using System;
using System.Collections.Generic;

namespace Archlens.Application;

public static class DependencyAggregator
{
    private static readonly StringComparer Cmp = StringComparer.OrdinalIgnoreCase;

    public static void RecomputeAggregates(DependencyGraph root)
    {
        _ = Fold(root);
    }

    private static Dictionary<string, int> Fold(DependencyGraph node)
    {
        if (node is DependencyGraphLeaf)
        {
            return new Dictionary<string, int>(node.GetDependencies(), Cmp);
        }

        var agg = new Dictionary<string, int>(Cmp);

        foreach (var child in node.GetChildren())
        {
            var childAgg = Fold(child);

            foreach (var (dep, count) in childAgg)
            {
                if (IsInternal(node, dep)) continue;
                agg[dep] = agg.TryGetValue(dep, out var cur) ? cur + count : count;
            }
        }

        if (node is DependencyGraphNode n)
            n.ReplaceDependencies(agg);

        return agg;
    }

    private static bool IsInternal(DependencyGraph node, string dep)
    {
        var ns = node.Path.Replace("./", String.Empty).Replace('/', '.');
        if (dep.Equals(ns) || String.IsNullOrEmpty(ns)) return true;
        return dep.StartsWith(ns, true, System.Globalization.CultureInfo.InvariantCulture);
    }
}

