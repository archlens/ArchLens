using Archlens.Application;
using Archlens.Domain.Models.Records;
using Archlens.Infra;
using Archlens.Infra.Factories;
using System;
using System.IO;
using System.Threading.Tasks;

namespace Archlens.CLI;

public class Program
{
    public static async Task Main(string[] args)
    {
        try
        {
            var options = await GetOptions(args[0].Trim());

            var snapshotManager = SnapsnotManagerFactory.SelectSnapshotManager(options);
            var parser = DependencyParserFactory.SelectDependencyParser(options);
            var renderer = RendererFactory.SelectRenderer(options.Format);

            var rendererService = new RendererService(options, snapshotManager, parser, renderer);

            await rendererService.RenderDependencyGraphAsync();
        }
        catch (Exception e)
        {
            Console.WriteLine($"EXCEPTION: {e.Message}\n{e.StackTrace}");
        }

    }

    private async static Task<Options> GetOptions(string args)
    {
        var configPath = args.Length > 0 ? args : FindConfigFile("archlens.json");

        var configManager = new ConfigManager(configPath);

        return await configManager.LoadAsync();
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
