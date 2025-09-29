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
    class DependencyGraphBuilder(string projectName, string root, List<string> excludes)
    {
        private string _root = root;
        private List<string> _excludes = excludes;

        // Start graph build
        public DependencyGraph GetGraph()
        {
            string[] dir = Directory.GetDirectories(_root);

            Node graph = new() { name = "root", children = [], dependencies = [] };

            BuildGraph(dir, graph);

            return graph;
        }


        // Recursively build graph
        public void BuildGraph(string[] directories, Node graph)
        {
            List<Node> children = [];

            foreach (var directory in directories)
            {
                if (_excludes.Exists(directory.Contains)) continue;

                Node node = new() { name = directory.Split("\\").Last(), children = [], dependencies = [] };

                string[] files = Directory.GetFiles(directory);

                foreach (string file in files)
                {
                    if (file.EndsWith(".cs"))
                    {
                        Leaf child = CreateLeaf(file);
                        node.AddChild(child);
                        foreach (var dep in child.dependencies)
                        {
                            node.AddDependency(dep, 1);
                        }
                    }
                }

                BuildGraph(Directory.GetDirectories(directory), node);

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