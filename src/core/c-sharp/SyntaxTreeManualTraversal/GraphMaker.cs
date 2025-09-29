using System.Collections.Generic;
using System.IO;
using System.Linq;
using SyntaxTreeManualTraversal.DependencyGraph;

namespace SyntaxTreeManualTraversal
{
    class GraphMaker(int depth, string projectName, string root, List<string> excludes)
    {
        private int _depth = depth;
        private string _root = root;
        private List<string> _excludes = excludes;
        private UsingMapper mapper = new UsingMapper(projectName);

        public DependencyGraph.DependencyGraph GetGraph()
        {
            string[] dir = Directory.GetDirectories(_root);

            Node graph = new() { name = "root", children = [] };

            MapFiles(dir, _depth, graph);

            return graph;
        }

        public void MapFiles(string[] dir, int depth, Node graph)
        {
            List<Node> children = [];

            foreach (var item in dir)
            {
                if (_excludes.Exists(item.Contains)) continue;

                Node node = new() { name = item.Split("\\").Last(), children = [] };

                string[] files = Directory.GetFiles(item);
                var dirChildren = mapper.MapFiles(files);

                node.AddChildren(dirChildren);

                if (depth > 0)
                {
                    MapFiles(Directory.GetDirectories(item), depth--, node);
                }

                children.Add(node);
            }

            graph.AddChildren(children);
        }
    }
}