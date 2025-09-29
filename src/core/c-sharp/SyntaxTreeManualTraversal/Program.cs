using System;
using System.Collections.Generic;

namespace SyntaxTreeManualTraversal
{
    internal class Program
    {
        //Config stuff
        static int depth = 2;
        static string projectName = "MilanoProject";
        static string root = "C://Users//lotte//Skrivebord//Repos//hygge-projekter//MilanoProject";
        static List<string> excludes = [".", "bin", "node_modules", "ClientApp", "obj", "Pages"];

        static void Main(string[] args)
        {
            var gm = new GraphMaker(depth, projectName, root, excludes);

            Console.WriteLine(gm.GetGraph().ToString());
        }
    }

}
