using Archlens.Domain.Utils;
using System;
using System.Collections;
using System.Collections.Generic;
using System.Linq;

namespace Archlens.Domain.Models;

public class DependencyGraph(string _projectRoot) : IEnumerable<DependencyGraph>
{
    private readonly string _path;
    private IDictionary<string, int> _dependencies { get; init; } = new Dictionary<string, int>();

    required public string Name { get; init; }
    required public string Path 
    {
        get => _path;
        init { _path = PathNormaliser.NormalisePath(_projectRoot, value); }
    }

    required public DateTime LastWriteTime { get; init; } = DateTime.UtcNow;
    
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

    public virtual DependencyGraph GetChild(string path) =>  GetChildren().Where(child => child.Path == path).FirstOrDefault();

    public virtual IReadOnlyList<DependencyGraph> GetChildren() => [];
    public override string ToString() => Name;
    public virtual string ToJson() => "";

    public virtual List<string> ToPlantUML(bool diff) => [];


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

public class DependencyGraphNode(string projectRoot) : DependencyGraph(projectRoot)
{
    private List<DependencyGraph> _children { get; init; } = [];
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
        var deps = child.GetDependencies().Keys;
        if (deps.Count > 0)
        {
            var ownedChildNames = _children.Select(c => c.Name).Where(ns => !string.IsNullOrEmpty(ns));

            if (ownedChildNames.Any())
            {
                foreach (var dep in deps)
                {
                    var isInternalDep = ownedChildNames.Any(cn => dep.Contains(cn, StringComparison.Ordinal));
                    if (!isInternalDep)
                        AddDependency(dep);
                }
            } else
            {
                AddDependencyRange([.. deps]);
            }
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
}

public class DependencyGraphLeaf(string projectRoot) : DependencyGraph(projectRoot)
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
}
