using System.Collections.Generic;
using System.Linq;

namespace Archlens.Domain.Models;
public class DependencyGraph
{
    public string Name { get; init; }

    override public string ToString()
    {
        return Name;
    }

    public virtual string ToJson()
    {
        return "{}";
    }

    public virtual List<string> packages()
    {
        return [];
    }
}

class Node : DependencyGraph
{
    public List<DependencyGraph> Children { get; init; }
    public Dictionary<string, List<DependencyGraph>> Dependencies { get; init; }

    public void AddChildren(IEnumerable<DependencyGraph> childr)
    {
        Children.AddRange(childr);
    }

    public void AddChild(DependencyGraph child)
    {
        Children.Add(child);
    }

    public void AddDependency(string dep, DependencyGraph child)
    {
        if (Dependencies.ContainsKey(dep))
            Dependencies[dep].Add(child);
        else
            Dependencies.Add(dep, [child]);
    }

    override public string ToString()
    {
        string res = Name;
        if (Dependencies.Keys.Count > 0)
            res += " (" + Dependencies.Values.Count + ")";
        else
            res += " (0)";

        foreach (var c in Children)
        {
            res += "\n \t";
            res += c.ToString();
        }

        return res;
    }

    override public string ToJson()
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

    override public List<string> packages()
    {
        List<string> res = [];
        foreach (var package in Children)
        {
            res.Add(Name);
            res.AddRange(package.packages());
        }

        return res;
    }
}

class Leaf : DependencyGraph
{
    public List<string> Dependencies { get; init; }

    override public string ToString()
    {
        string res = "\t" + Name;
        foreach (var d in Dependencies)
        {
            res += "\n \t \t --> " + d;
        }
        return res;
    }

    override public string ToJson()
    {
        return "";
    }

    override public List<string> packages()
    {
        return [];
    }
}
