using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;
using SyntaxTreeManualTraversal.DependencyGraph;


namespace SyntaxTreeManualTraversal
{
    class UsingMapper
    {
        private string _programName;
        public Dictionary<string, List<string>> mapping { get; }

        public UsingMapper(string programName)
        {
            _programName = programName;
            mapping = new Dictionary<string, List<string>>();
        }

        public List<Leaf> MapFiles(string[] filenames)
        {
            List<Leaf> children = [];

            foreach (string file in filenames)
            {
                if (file.EndsWith(".cs"))
                    children.Add(MapFile(file));
            }

            return children;
        }


        private Leaf MapFile(string filename)
        {
            string lines = "";
            string namesp = filename.Split('\\').Last(); //Use filename as namespace, if none is included
            List<string> usings = [];

            try
            {
                StreamReader sr = new StreamReader(filename);

                string line = sr.ReadLine();

                while (line != null)
                {
                    if (line.StartsWith("namespace"))
                    {
                        namesp = line.Replace("namespace", "").Replace(" ", "");
                    }
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

            var collector = new UsingCollector(_programName);
            collector.Visit(root);

            foreach (var directive in collector.Usings)
            {
                usings.Add(directive.Name.ToString());
            }

            if (mapping.ContainsKey(namesp))
                mapping[namesp].AddRange(usings);
            else
                mapping.Add(namesp, usings);

            Leaf leaf = new() { dependencies = usings, name = filename.Split("\\").Last() };

            return leaf;
        }
    }
}