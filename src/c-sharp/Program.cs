using Archlens.Application;
using System;
using System.IO;
using System.Threading.Tasks;

namespace Archlens.Main;

public class Program
{
    public static async Task Main(string[] args)
    {
        try
        {
            var configPath = args.Length > 0 ? args[0].Trim() : FindConfigFile("archlens.json");

            var rendererService = new RendererService(configPath);

            await rendererService.RenderDependencyGraphAsync();
        }
        catch (Exception e)
        {
            Console.WriteLine($"EXCEPTION: {e.Message}\n{e.StackTrace}");
        }

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
