using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;
using SyntaxTreeManualTraversal.Model;

namespace SyntaxTreeManualTraversal
{
    // Builds a dependency graph
    class GraphBuilder(string projectName, string root, List<string> excludes)
    {
        private string _root = root;
        private List<string> _excludes = excludes;

        // Start graph build
        public DependencyGraph GetGraph()
        {
            string[] dir = Directory.GetDirectories(_root);

            Node graph = new() { name = "root", children = [] };

            BuildGraph(dir, graph);

            return graph;
        }


        // Recursively build graph
        public void BuildGraph(string[] dir, Node graph)
        {
            List<Node> children = [];

            foreach (var item in dir)
            {
                if (_excludes.Exists(item.Contains)) continue;

                Node node = new() { name = item.Split("\\").Last(), children = [] };

                string[] files = Directory.GetFiles(item);

                List<Leaf> dirChildren = [];

                foreach (string file in files)
                {
                    if (file.EndsWith(".cs"))
                        dirChildren.Add(CreateLeaf(file));
                }

                node.AddChildren(dirChildren);

                BuildGraph(Directory.GetDirectories(item), node);

                children.Add(node);
            }

            graph.AddChildren(children);
        }


        // Create and return leaf with dependencies
        private Leaf CreateLeaf(string filename)
        {
            string lines = "";
            List<string> usings = [];

            try
            {
                StreamReader sr = new StreamReader(filename);

                string line = sr.ReadLine();

                while (line != null)
                {
                    lines += "\n" + line;
                    line = sr.ReadLine();
                }

                sr.Close();
            }
            catch (Exception e)
            {
                Console.WriteLine("Exception: " + e.Message);
            }

            SyntaxTree tree = CSharpSyntaxTree.ParseText(lines);
            CompilationUnitSyntax root = tree.GetCompilationUnitRoot();

            var collector = new UsingCollector(projectName);
            collector.Visit(root);

            foreach (var directive in collector.Usings)
            {
                usings.Add(directive.Name.ToString());
            }

            Leaf leaf = new() { dependencies = usings, name = filename.Split("\\").Last() };

            return leaf;
        }
    }
}