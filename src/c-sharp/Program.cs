using System;
using System.Threading.Tasks;
using Archlens.Application;
using Archlens.Domain;

namespace Archlens;

internal class Program
{
    static async Task Main(string[] args)
    {
        var configPath = args.Length > 0 ? args[0].Trim() : "../../../archfig.json" ;

        var configManager = new ConfigManager(configPath);

        var rendererService = new RendererService(configManager);

        Console.WriteLine(await rendererService.RenderDependencyGraphAsync());
    }
}
