using System.Collections.Generic;

namespace SyntaxTreeManualTraversal.DependencyGraph
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

        override public string ToString()
        {
            string res = name;
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

