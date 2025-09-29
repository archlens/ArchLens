using System.Collections.Generic;
using System.Linq;

namespace SyntaxTreeManualTraversal.Model
{
    class DependencyGraph
    {
        public string name { get; init; }

        override public string ToString()
        {
            return name;
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
        public List<DependencyGraph> children { get; init; }
        public Dictionary<string, List<DependencyGraph>> dependencies { get; init; }

        public void AddChildren(IEnumerable<DependencyGraph> childr)
        {
            children.AddRange(childr);
        }

        public void AddChild(DependencyGraph child)
        {
            children.Add(child);
        }

        public void AddDependency(string dep, DependencyGraph child)
        {
            if (dependencies.ContainsKey(dep))
                dependencies[dep].Add(child);
            else
                dependencies.Add(dep, [child]);
        }

        override public string ToString()
        {
            string res = name;
            if (dependencies.Keys.Count > 0)
                res += " (" + dependencies.Values.Count + ")";
            else
                res += " (0)";

            foreach (var c in children)
            {
                res += "\n \t";
                res += c.ToString();
            }

            return res;
        }

        override public string ToJson()
        {
            var str = "";
            if (dependencies.Keys.Count > 0)
            {
                for (int i = 0; i < dependencies.Keys.Count; i++)
                {
                    var dep = dependencies.Keys.ElementAt(i);
                    var relations = "";
                    string comma;
                    for (int j = 0; j < dependencies[dep].Count; j++)
                    {
                        var rel = dependencies[dep][j];
                        if (j < dependencies[dep].Count - 1) comma = ",";
                        else comma = "";

                        relations +=
                        $$"""
                        {
                                    "from_file": {
                                        "name": "{{rel.name}}",
                                        "path": "{{rel.name}}"
                                    },
                                    "to_file": {
                                        "name": "{{dep}}",
                                        "path": "{{dep}}"
                                    }
                                }{{comma}}
                        """;
                    }

                    if (i < dependencies.Keys.Count - 1 || i == 0) comma = ",";
                    else comma = "";

                    str +=
                    $$"""
                        {
                            "state": "NEUTRAL",
                            "fromPackage": "{{name}}",
                            "toPackage": "{{dep}}",
                            "label": "{{dependencies[dep].Count}}",
                            "relations": [
                                {{relations}}
                            ]
                        }{{comma}}

                    """;
                }

            }

            for (int c = 0; c < children.Count; c++)
            {
                var child = children[c];
                var childJson = child.ToJson();
                str += childJson;
                if ((c < children.Count - 1 || c == 0) && childJson != "") str += ",";
            }

            return str;

        }

        override public List<string> packages()
        {
            List<string> res = [];
            foreach (var package in children)
            {
                res.Add(name);
                res.AddRange(package.packages());
            }

            return res;
        }
    }

    class Leaf : DependencyGraph
    {
        public List<string> dependencies { get; init; }

        override public string ToString()
        {
            string res = "\t" + name;
            foreach (var d in dependencies)
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
}

