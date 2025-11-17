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
    private Dictionary<string, int> _dependencies { get; init; } = [];
    
    required public DateTime LastWriteTime 
    { 
        get => _lastWriteTime;
        init { _lastWriteTime = DateTimeNormaliser.NormaliseUTC(value); }
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
        if (_children.Any(c => ReferenceEquals(c, child) || c.Path == child.Path))
            return;
        _children.Add(child);
    }

    internal void ReplaceDependencies(IDictionary<string, int> newDeps)
    {
        var dict = GetDependencies();
        dict.Clear();
        foreach (var kv in newDeps)
            dict[kv.Key] = kv.Value;
    }

    public override string ToString()
    {
        string res = Name + $" ({GetDependencies()})";
        foreach (var c in _children)
            res += "\n \t" + c;
        return res;
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
}
