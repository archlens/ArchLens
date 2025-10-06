using System;
using System.Collections.Generic;
using System.IO;
using System.Threading.Tasks;
using Archlens.Application;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra;

namespace Archlens;

internal class Program
{
    //Config stuff
    static string projectName = "MilanoProject";
    static string root = "C://Users//lotte//Skrivebord//Repos//hygge-projekter//MilanoProject";
    static List<string> excludes = [".", "bin", "node_modules", "ClientApp", "obj", "Pages", "Properties", "wwwroot"];

    static async Task Main(string[] args)
    {
        var options = new Options(root, projectName, Language.CSharp, Baseline.Local, RenderFormat.Json, excludes, [".cs"]);

        var gm = new DependencyGraphBuilder(new CsharpDependencyParser(options), options);

        var graph = await gm.GetGraphAsync(root, Directory.GetDirectories(root));

        File.WriteAllText(@"C:\Users\lotte\Skrivebord\ITU\CS3\Research-project\ArchLens\src\c-sharp\graph-json.json", new JsonRenderer().RenderGraph(graph, options));

        Console.WriteLine(graph.ToString());
    }
}
