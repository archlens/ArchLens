using System.Threading.Tasks;
using Archlens.Application;
using Archlens.Domain;
using System;
using System.IO;
using System.Threading.Tasks;

namespace Archlens.Main;

public class Program
{
    public static async Task Main(string[] args)
    {
        var configPath = args.Length > 0 ? args[0].Trim() : FindConfigFile("archlens.json");
        
        var configManager = new ConfigManager(configPath);

        var rendererService = new RendererService(configManager);

        await rendererService.RenderDependencyGraphAsync();
    }

    private static string FindConfigFile(string fileName)
    {
        var dir = AppContext.BaseDirectory;

        while (!string.IsNullOrEmpty(dir))
        {
            var candidate = Path.Combine(dir, fileName);
            if (File.Exists(candidate))
                return candidate;

            dir = Directory.GetParent(dir)?.FullName;
        }

        throw new FileNotFoundException($"Could not find '{fileName}' starting from '{AppContext.BaseDirectory}'.");
    }
}
