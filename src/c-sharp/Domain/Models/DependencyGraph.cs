using System;
using System.Collections;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Text.Json;

namespace Archlens.Domain.Models;

public class DependencyGraph : IEnumerable<DependencyGraph>
{
    public string Name { get; init; }
    public DateTime LastWriteTime { get; init; } = DateTime.UtcNow;
    private IDictionary<string, int> _dependencies { get; init; } = new Dictionary<string, int>();

    public IDictionary<string, int> GetDependencies() => _dependencies;

    public void AddDependency(string depPath)
    {
        if (_dependencies.TryGetValue(depPath, out int value))
            _dependencies[depPath] = ++value;
        else
            _dependencies[depPath] = 1;
    }

    public void AddDependencyRange(IReadOnlyList<string> depPaths)
    {
        foreach (var depPath in depPaths)
        {
            AddDependency(depPath);
        }
    }

    public virtual DependencyGraph GetChild(string name) =>  GetChildren().Where(child => child.Name == name).FirstOrDefault();

    public virtual IReadOnlyList<DependencyGraph> GetChildren() => [];
    public override string ToString() => Name;
    public virtual string ToJson() => "";

    public virtual string Serialize() => "{}";

    public virtual List<string> ToPlantUML(bool diff) => [];

    public virtual List<string> Packages() => [];

    public static DependencyGraph Deserialize(string json)
    {
        using var doc = JsonDocument.Parse(json);
        var rootEl = doc.RootElement;

        return rootEl.ValueKind switch
        {
            JsonValueKind.Object => ParseNode(rootEl),
            _ => throw new InvalidOperationException("Expected a JSON object or array at root.")
        };
    }

    private static DependencyGraph ParseNode(JsonElement jsonNode)
    {
        var name = jsonNode.TryGetProperty("name", out var nEl) ? nEl.GetString() : String.Empty;
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
                children.Add(ParseNode(child));
        }

        if (children.Count <= 0)
        {
            var leaf = new DependencyGraphLeaf
            {
                Name = name,
                LastWriteTime = lastWrite
            };
            foreach (var (path, _) in depKeys)
                leaf.AddDependency(path);
            return leaf;
        }
        else
        {
            var node = new DependencyGraphNode
            {
                Name = name,
                LastWriteTime = lastWrite
            };
            foreach (var (path, count) in depKeys)
                for (int i = 0; i < count; i++) node.AddDependency(path);

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
            if (DateTime.TryParse(s, CultureInfo.InvariantCulture, DateTimeStyles.AssumeUniversal | DateTimeStyles.AdjustToUniversal, out var dt))
                return DateTime.SpecifyKind(dt, DateTimeKind.Utc);
            if (DateTime.TryParse(s, out dt))
                return DateTime.SpecifyKind(dt, DateTimeKind.Utc);
            return DateTime.UtcNow;
        }

        if (tEl.ValueKind == JsonValueKind.Number && tEl.TryGetInt64(out var ms))
            return DateTimeOffset.FromUnixTimeMilliseconds(ms).UtcDateTime;

        if (tEl.TryGetDateTime(out var direct))
            return DateTime.SpecifyKind(direct, DateTimeKind.Utc);

        return DateTime.UtcNow;
    }

    IEnumerator IEnumerable.GetEnumerator() => GetEnumerator();

    public IEnumerator<DependencyGraph> GetEnumerator()
    {
        return Traverse(this).GetEnumerator();

        static IEnumerable<DependencyGraph> Traverse(DependencyGraph node)
        {
            yield return node;
            foreach (var child in node.GetChildren())
            {
                foreach (var desc in Traverse(child))
                    yield return desc;
            }
        }
    }
}

public class DependencyGraphNode : DependencyGraph
{
    private List<DependencyGraph> _children { get; init; } = [];
    private List<string> _packages;
    public override IReadOnlyList<DependencyGraph> GetChildren() => _children;
    public void AddChildren(IEnumerable<DependencyGraph> childr)
    {
        foreach (var child in childr)
        {
            AddChild(child);
        }
    }
    public void AddChild(DependencyGraph child)
    {
        if (child.GetDependencies().Count > 0)
        {
            var ownedChildNames = _children.Select(c => c.Name);

            var uniqueKeys = child.GetDependencies().Keys
                .Where(k => !ownedChildNames.Any(n => k.Contains(n, StringComparison.Ordinal)));

            AddDependencyRange([.. uniqueKeys]);
        }
        _children.Add(child);
    }

    public override string ToString()
    {
        string res = Name + $" ({GetDependencies()})";
        foreach (var c in _children)
            res += "\n \t" + c;
        return res;
    }

    public override string ToJson()
    {
        var str = "";
        var dependencies = GetDependencies();
        if (dependencies.Keys.Count > 0)
        {
            for (int i = 0; i < dependencies.Keys.Count; i++)
            {
                var dep = dependencies.Keys.ElementAt(i);
                var relations = "";
                for (int j = 0; j < dependencies[dep]; j++)
                {
                    var rel = dependencies[dep];
                    if (j > 0) relations += ",\n";

                    relations +=
                    $$"""
                            {
                                "from_file": {
                                    "name": "{{dependencies.Keys}}",
                                    "path": "{{dependencies.Keys}}"
                                },
                                "to_file": {
                                    "name": "{{dep}}",
                                    "path": "{{dep}}"
                                }
                            }
                    """;
                }

                if (i > 0) str += ",\n";

                str +=
                $$"""
                    {
                        "state": "NEUTRAL",
                        "fromPackage": "{{Name}}",
                        "toPackage": "{{dep}}",
                        "label": "{{dependencies[dep]}}",
                        "relations": [
                            {{relations}}
                        ]
                    }
                """;
            }

        }

        var children = GetChildren();
        for (int c = 0; c < children.Count; c++)
        {
            var child = children[c];
            var childJson = child.ToJson();
            if (c > 0 && childJson != "" && !childJson.StartsWith(',') && str != "")
                str += ",\n";

            str += childJson;
        }

        return str;

    }

    public override string Serialize()
    {
        var dependencies = GetDependencies();
        var depsJson = dependencies.Any() ? $"\n{string.Join(",\n", dependencies.Select(d => $"{{ \"{d.Key}\": {d.Value} }}"))}\n" : "";

        var children = GetChildren();
        var childrenJson = children.Any() ? $"\n{string.Join(",\n", GetChildren().Select(c => c.Serialize()))}\n" : "";

        return $$"""
                {
                    "name": "{{Name}}",
                    "lastWriteTime": "{{LastWriteTime}}",
                    "state": "NEUTRAL",
                    "relations": 
                    [ {{depsJson}} ],
                    "children": 
                    [{{childrenJson}}]
                }
                """;
    }

    public override List<string> ToPlantUML(bool diff)
    { //TODO: Add color depending on diff
        string package = $"package \"{Name}\" as {Name} {{ \n";

        List<string> puml = [];

        foreach (var child in _children)
        {
            string childName = child.Name.Replace(" ", "-");

            if (child is DependencyGraphLeaf)
            {
                package += $"\n [{childName}]";
                var childList = child.ToPlantUML(diff);
                puml.AddRange(childList);
            }
            else
            {
                var childList = child.ToPlantUML(diff);
                var c = childList.Last(); //last is the package declaration, which we want to be added here
                package += $"\n{c}\n";
                childList.Remove(c);
                puml.AddRange(childList);
            }
        }
        package += "\n}\n";
        puml.Add(package);
        return puml;
    }

    public override List<string> Packages()
    {
        if (_packages != null) return _packages;

        List<string> res = [];
        foreach (var package in _children)
        {
            res.Add(Name);
            res.AddRange(package.Packages());
        }

        _packages = res;
        return res;
    }
}

public class DependencyGraphLeaf : DependencyGraph
{
    public override string ToString()
    {
        var res = "\t" + Name;
        foreach (var d in GetDependencies().Keys)
            res += "\n \t \t --> " + d;
        return res;
    }

    public override List<string> ToPlantUML(bool diff)
    { //TODO: diff
        List<string> puml = [];

        foreach (var dep in GetDependencies().Keys)
        {
            puml.Add($"\n\"{Name}\"-->{dep}"); //package alias
        }
        return puml;
    }

    public override string Serialize() => "";
    public override List<string> Packages() => [];
}
