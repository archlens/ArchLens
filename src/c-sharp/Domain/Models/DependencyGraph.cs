using System;
using System.Collections;
using System.Collections.Generic;
using System.Linq;

namespace Archlens.Domain.Models;
public class DependencyGraph : IEnumerable<DependencyGraph>
{
    public string Name { get; init; }
    public DateTime LastWriteTime { get; init; }

    public virtual DependencyGraph GetChild(string name)
    {
        return GetChildren().Where(child => child.Name == name).FirstOrDefault();
    }
    public override string ToString() => Name;

    public virtual string ToJson() => "{}";

    public virtual List<string> Packages() => [];

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

    IEnumerator IEnumerable.GetEnumerator() => GetEnumerator();

    protected virtual IReadOnlyList<DependencyGraph> GetChildren() =>
        [];
    }

public class Node : DependencyGraph
    {
    public List<DependencyGraph> Children { get; init; } = [];
    public Dictionary<string, List<DependencyGraph>> Dependencies { get; init; } = [];

    protected override IReadOnlyList<DependencyGraph> GetChildren() => Children;

    public void AddChildren(IEnumerable<DependencyGraph> childr) => Children.AddRange(childr);
    public void AddChild(DependencyGraph child) => Children.Add(child);

    public void AddDependency(string dep, DependencyGraph child)
    {
        if (Dependencies.TryGetValue(dep, out List<DependencyGraph> value))
            value.Add(child);
        else
            Dependencies.Add(dep, [child]);
    }

    public override string ToString()
    {
        string res = Name + $" ({Dependencies.Values.Sum(l => l.Count)})";
        foreach (var c in Children)
            res += "\n \t" + c;
        return res;
    }

    public override string ToJson()
    {
        var str = "";
        if (Dependencies.Keys.Count > 0)
        {
            for (int i = 0; i < Dependencies.Keys.Count; i++)
            {
                var dep = Dependencies.Keys.ElementAt(i);
                var relations = "";
                string comma;
                for (int j = 0; j < Dependencies[dep].Count; j++)
                {
                    var rel = Dependencies[dep][j];
                    if (j < Dependencies[dep].Count - 1) comma = ",";
                    else comma = "";

                    relations +=
                    $$"""
                    {
                                "from_file": {
                                    "name": "{{rel.Name}}",
                                    "path": "{{rel.Name}}"
                                },
                                "to_file": {
                                    "name": "{{dep}}",
                                    "path": "{{dep}}"
                                }
                            }{{comma}}
                    """;
                }

                if (i < Dependencies.Keys.Count - 1 || i == 0) comma = ",";
                else comma = "";

                str +=
                $$"""
                    {
                        "state": "NEUTRAL",
                        "fromPackage": "{{Name}}",
                        "toPackage": "{{dep}}",
                        "label": "{{Dependencies[dep].Count}}",
                        "relations": [
                            {{relations}}
                        ]
                    }{{comma}}

                """;
            }

        }

        for (int c = 0; c < Children.Count; c++)
        {
            var child = Children[c];
            var childJson = child.ToJson();
            str += childJson;
            if ((c < Children.Count - 1 || c == 0) && childJson != "") str += ",";
        }

        return str;

    }

    public override List<string> Packages()
    {
        List<string> res = [];
        foreach (var package in Children)
        {
            res.Add(Name);
            res.AddRange(package.Packages());
        }

        return res;
    }
}

public class Leaf : DependencyGraph
{
    public IReadOnlyList<string> Dependencies { get; init; } = [];

    public override string ToString()
    {
        var res = "\t" + Name;
        foreach (var d in Dependencies)
            res += "\n \t \t --> " + d;
        return res;
    }

    public override string ToJson() => "";
    public override List<string> Packages() => [];
}
