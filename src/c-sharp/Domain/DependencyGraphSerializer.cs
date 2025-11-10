
using Archlens.Domain.Models;
using Archlens.Domain.Utils;
using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Text.Json;
namespace Archlens.Domain;

public static class DependencyGraphSerializer
{
    public static string Serialize(DependencyGraph graph)
    {
        return graph switch
        {
            DependencyGraphNode node => SerializeNode(node),
            DependencyGraphLeaf leaf => SerializeLeaf(leaf),
            _ => throw new InvalidOperationException("Unknown DependencyGraph type"),
        };
    }

    public static DependencyGraph Deserialize(string json)
    {
        using var doc = JsonDocument.Parse(json);
        var rootEl = doc.RootElement;
        var rootPath = rootEl.TryGetProperty("path", out var pEl) ? pEl.GetString() : String.Empty;

        return rootEl.ValueKind switch
        {
            JsonValueKind.Object => ParseNode(rootEl, rootPath),
            _ => throw new InvalidOperationException("Expected a JSON object or array at root.")
        };
    }

    private static string SerializeNode(DependencyGraphNode node)
    {
        var childrenSerialized = new List<string>();

        foreach (var child in node.GetChildren())
            childrenSerialized.Add(Serialize(child));

        var dependencies = node.GetDependencies();
        var depsSerialized = dependencies.Count > 0 ? $"\n{string.Join(",\n", dependencies.Select(d => $"{{ \"{d.Key}\": {d.Value} }}"))}\n" : "";
        return $$"""
                {
                    "type": "node",
                    "name": "{{node.Name}}",
                    "path": "{{node.Path}}",
                    "lastWriteTime": "{{node.LastWriteTime}}",
                    "relations": [{{depsSerialized}}],
                    "children": [
                        {{string.Join(",\n", childrenSerialized)}}
                    ]
                }
                """;
    }

    private static string SerializeLeaf(DependencyGraphLeaf leaf)
    {
        var dependencies = leaf.GetDependencies();
        var depsJson = dependencies.Count > 0 ? $"\n{string.Join(",\n", dependencies.Select(d => $"{{ \"{d.Key}\": {d.Value} }}"))}\n" : "";
        return $$"""
                {
                    "type": "leaf",
                    "name": "{{leaf.Name}}",
                    "path": "{{leaf.Path}}",
                    "lastWriteTime": "{{leaf.LastWriteTime}}",
                    "relations": [{{depsJson}}]
                }
                """;
    }

    private static DependencyGraph ParseNode(JsonElement jsonNode, string rootPath)
    {
        var name = jsonNode.TryGetProperty("name", out var nEl) ? nEl.GetString() : String.Empty;
        var nameSpace = jsonNode.TryGetProperty("nameSpace", out var nsEl) ? nsEl.GetString() : String.Empty;
        var path = jsonNode.TryGetProperty("path", out var pEl) ? pEl.GetString() : String.Empty;
        var lastWrite = ReadDateTime(jsonNode);

        var depKeys = new List<(string path, int count)>();

        if (jsonNode.TryGetProperty("relations", out var jsonDeps) && jsonDeps.ValueKind == JsonValueKind.Array)
        {
            foreach (var depPair in jsonDeps.EnumerateArray())
            {
                if (depPair.ValueKind == JsonValueKind.Object)
                {
                    foreach (var prop in depPair.EnumerateObject())
                    {
                        var key = prop.Name;
                        int count =
                            prop.Value.ValueKind == JsonValueKind.Number && prop.Value.TryGetInt32(out var n) ? n :
                            prop.Value.ValueKind == JsonValueKind.String && int.TryParse(prop.Value.GetString(), NumberStyles.Integer, CultureInfo.InvariantCulture, out var s) ? s :
                            -998;
                        depKeys.Add((key, count));
                    }
                }
                else if (depPair.ValueKind == JsonValueKind.String)
                {
                    depKeys.Add((depPair.GetString()!, -999));
                }
            }
        }

        var children = new List<DependencyGraph>();
        if (jsonNode.TryGetProperty("children", out var jsonChildren) && jsonChildren.ValueKind == JsonValueKind.Array)
        {
            foreach (var child in jsonChildren.EnumerateArray())
                children.Add(ParseNode(child, rootPath));
        }

        if (children.Count <= 0)
        {
            var leaf = new DependencyGraphLeaf(rootPath)
            {
                Name = name,
                Path = path,
                LastWriteTime = lastWrite
            };
            foreach (var (dep, _) in depKeys)
                leaf.AddDependency(dep);
            return leaf;
        }
        else
        {
            var node = new DependencyGraphNode(rootPath)
            {
                Name = name,
                Path = PathNormaliser.NormalisePath(rootPath, path),
                LastWriteTime = lastWrite
            };
            foreach (var (dep, count) in depKeys)
                for (int i = 0; i < count; i++) node.AddDependency(dep);

            node.AddChildren(children);
            return node;
        }
    }

    private static DateTime ReadDateTime(JsonElement el)
    {
        if (!el.TryGetProperty("lastWriteTime", out var tEl))
            return DateTime.UtcNow;

        if (tEl.ValueKind == JsonValueKind.String)
        {
            var s = tEl.GetString();
            if (DateTime.TryParse(s, out var dt))
                return DateTime.SpecifyKind(dt, DateTimeKind.Utc);
            return DateTime.UtcNow;
        }

        if (tEl.ValueKind == JsonValueKind.Number && tEl.TryGetInt64(out var ms))
            return DateTimeOffset.FromUnixTimeMilliseconds(ms).UtcDateTime;

        if (tEl.TryGetDateTime(out var direct))
            return DateTime.SpecifyKind(direct, DateTimeKind.Utc);

        return DateTime.UtcNow;
    }
}
