using Archlens.Domain.Utils;
using System;
using System.Collections;
using System.Collections.Generic;
using System.Linq;

namespace Archlens.Domain.Models;

public class DependencyGraph(string _projectRoot) : IEnumerable<DependencyGraph>
{
    private readonly DateTime _lastWriteTime;
    private readonly string _path;
    private IDictionary<string, int> _dependencies { get; init; } = new Dictionary<string, int>();
    
    required public DateTime LastWriteTime 
    { 
        get => _lastWriteTime;
        init { _lastWriteTime = NormaliseUTC(value); }
    }

    required public string Name { get; init; }
    required public string Path 
    {
        get => _path;
        init { _path = PathNormaliser.NormalisePath(_projectRoot, value); }
    }
    
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

    public virtual DependencyGraph GetChild(string path) => GetChildren().Where(child => child.Path == path).FirstOrDefault();

    public virtual IReadOnlyList<DependencyGraph> GetChildren() => [];
    public override string ToString() => Name;

    public virtual List<string> ToPlantUML(bool diff, bool isRoot = true) => [];


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

    private static DateTime NormaliseUTC(DateTime time)
    {
        var convertedDate = DateTime.SpecifyKind(time, DateTimeKind.Utc);
        return convertedDate.ToLocalTime();
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
            }
            else
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
        var relations = "";
        var children = GetChildren();

        foreach (var (dep, count) in GetDependencies())
        {
            foreach (var child in children)
            {
                if (child.GetDependencies().TryGetValue(dep, out int val))
                {
                    var childName = child.Name.Replace("\\", ".");
                    if (!relations.EndsWith(",\n") && !string.IsNullOrEmpty(relations)) relations += ",\n";

                    relations +=
                        $$"""
                            {
                                "from_file": {
                                    "name": "{{childName}}",
                                    "path": "{{childName}}"
                                },
                                "to_file": {
                                    "name": "{{dep}}",
                                    "path": "{{dep}}"
                                }
                            }
                        """;
                }
            }

            if (!str.EndsWith(",\n") && !string.IsNullOrEmpty(str)) str += ",\n";

            str += //TODO: diff view (state)
                $$"""
                        {
                            "state": "NEUTRAL",
                            "fromPackage": "{{Name}}",
                            "toPackage": "{{dep}}",
                            "label": "{{count}}",
                            "relations": [
                                {{relations}}
                            ]
                        }
                    """;
        }

        return str;

    }


    public override List<string> ToPlantUML(bool diff, bool isRoot = true)
    {
        List<string> puml = [];

        if (isRoot)
        {
            foreach (var child in _children)
            {
                if (child is DependencyGraphNode)
                {
                    var childList = child.ToPlantUML(diff, false);

                    puml.AddRange(childList);
                }
            }
        }
        else
        {
            puml.Add($"package \"{Name.Replace("\\", ".")}\" as {Name.Replace("\\", ".")} {{ }}");

            foreach (var (dep, count) in GetDependencies())
            {
                var fromName = Name;
                var toName = dep.Split(".")[0];
                var existing = puml.Find(p => p.StartsWith($"{fromName}-->{toName} : "));
                if (string.IsNullOrEmpty(existing))
                    puml.Add($"{fromName}-->{toName} : {count}"); //TODO: Add color depending on diff
                else
                {
                    var existingCount = existing.Replace($"{fromName}-->{toName} : ", "");
                    var canParse = int.TryParse(existingCount, out var exCount);

                    if (!canParse) Console.WriteLine("Error parsing " + existingCount);

                    var newCount = canParse ? exCount + count : count;

                    puml.Remove(existing);
                    puml.Add($"{fromName}-->{toName} : {newCount}"); //TODO: Add color depending on diff
                }
            }
        }
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
