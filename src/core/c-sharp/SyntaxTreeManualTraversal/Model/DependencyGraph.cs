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
    }

    class Node : DependencyGraph
    {
        public List<DependencyGraph> children { get; init; }
        public Dictionary<string, int> dependencies { get; init; }

        public void AddChildren(IEnumerable<DependencyGraph> childr)
        {
            children.AddRange(childr);
        }

        public void AddChild(DependencyGraph child)
        {
            children.Add(child);
        }

        public void AddDependency(string dep, int count)
        {
            if (dependencies.ContainsKey(dep))
                dependencies[dep] += count;
            else
                dependencies.Add(dep, count);
        }

        override public string ToString()
        {
            string res = name;
            if (dependencies.Keys.Count > 0)
                res += " (" + dependencies.Values.Aggregate((i, j) => i + j) + ")";
            else
                res += " (0)";

            foreach (var c in children)
            {
                res += "\n \t";
                res += c.ToString();
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
    }
}

