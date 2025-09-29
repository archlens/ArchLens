using System;
using System.Collections.Generic;
using System.IO;

namespace SyntaxTreeManualTraversal
{
    internal class Program
    {
        //Config stuff
        static string projectName = "MilanoProject";
        static string root = "C://Users//lotte//Skrivebord//Repos//hygge-projekter//MilanoProject";
        static List<string> excludes = [".", "bin", "node_modules", "ClientApp", "obj", "Pages", "Properties", "wwwroot"];

        static void Main(string[] args)
        {
            var gm = new DependencyGraphBuilder(projectName, root, excludes);

            File.WriteAllText(@"C:\Users\lotte\Skrivebord\ITU\CS3\Research-project\ArchLens\src\core\c-sharp\SyntaxTreeManualTraversal\graph-json.json", GraphToJsonConverter.ConvertToJson(gm.GetGraph(), projectName));
        }
    }

}
