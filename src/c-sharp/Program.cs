using System;
using System.Collections.Generic;
using System.IO;
using System.Threading.Tasks;
using Archlens.Application;
using Archlens.Domain;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra;

namespace Archlens;

internal class Program
{
    //Config stuff
    //TODO: Use actual config instead
    static string projectName = "Archlens";
    static string root = Directory.GetCurrentDirectory();
    static string diagrams = Directory.GetParent(root).Parent.FullName + "/diagrams";
    static List<string> excludes = [".", "bin", "node_modules", "ClientApp", "obj", "Pages", "Properties", "wwwroot"];

    static async Task Main(string[] args)
    {
        var configPath = args[0].Trim();

        var configManager = new ConfigManager(configPath);

        var options = new Options(root, projectName, Language.CSharp, Baseline.Local, RenderFormat.PlantUML, excludes, [".cs"]);

        var gm = new DependencyGraphBuilder(new CsharpDependencyParser(options), options);

        var graph = await gm.GetGraphAsync(root, Directory.GetDirectories(root));

        if (options.Format == RenderFormat.Json)
            File.WriteAllText($@"{diagrams}\graph-json.json", new JsonRenderer().RenderGraph(graph, options));
        if (options.Format == RenderFormat.PlantUML)
            File.WriteAllText($@"{diagrams}\graph-puml.puml", new PlantUMLRenderer().RenderGraph(graph, options));
    }
}
