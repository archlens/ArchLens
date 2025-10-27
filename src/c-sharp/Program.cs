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
        var configPath = args.Length > 0 ? args[0].Trim() : "../../../archfig.json" ;

        var configManager = new ConfigManager(configPath);

        var rendererService = new RendererService(configManager);

        Console.WriteLine(await rendererService.RenderDependencyGraphAsync());
    }
}
