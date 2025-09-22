using System;
using System.IO;
using System.Linq;

namespace SyntaxTreeManualTraversal
{
    internal class Program
    {
        static int depth = 2;
        static UsingMapper mapper = new UsingMapper();

        static void Main(string[] args)
        {
            string root = "C://Users//lotte//Skrivebord//Repos//hygge-projekter//MilanoProject";

            string[] dir = Directory.GetDirectories(root);

            MapFiles(dir, depth);

            foreach (var item in mapper.GetMapping())
            {
                Console.WriteLine(item.Key);
                foreach (var val in item.Value)
                {
                    Console.WriteLine("--" + item.Key + " " + val);
                }
            }
        }

        static void MapFiles(string[] dir, int depth)
        {
            foreach (var item in dir)
            {
                if (item.Contains(".") || item.Contains("bin") || item.Contains("node_modules") || item.Contains("ClientApp")) continue;

                string[] files = Directory.GetFiles(item).Where(f => f.EndsWith(".cs")).ToArray();
                mapper.MapFiles(files);

                if (depth > 0)
                {
                    MapFiles(Directory.GetDirectories(item), depth--);
                }
            }
        }
    }
}
