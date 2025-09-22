using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;


namespace SyntaxTreeManualTraversal
{
    class UsingMapper
    {
        Dictionary<string, List<string>> mapping;

        public UsingMapper()
        {
            mapping = new Dictionary<string, List<string>>();
        }

        public Dictionary<string, List<string>> GetMapping()
        {
            return mapping;
        }

        public void MapFiles(string[] filenames)
        {
            foreach (string file in filenames)
            {
                MapFile(file);
            }
        }


        private void MapFile(string filename)
        {
            string lines = "";
            string namesp = filename.Split('\\').Last();
            List<string> usings = new List<string>();

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

            var collector = new UsingCollector();
            collector.Visit(root);

            foreach (var directive in collector.Usings)
            {
                usings.Add(directive.Name.ToString());
            }

            if (mapping.ContainsKey(namesp))
                mapping[namesp].AddRange(usings);
            else
                mapping.Add(namesp, usings);
        }
    }
}