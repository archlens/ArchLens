using System;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain;
using Archlens.Domain.Factories;

namespace Archlens.Application;

public sealed class RendererService(ConfigManager _config)
{

    public async Task<string> RenderDependencyGraphAsync(CancellationToken ct = default)
    {
        var options = await _config.LoadAsync(ct);

        var snapshotManager = SnapsnotManagerFactory.SelectSnapshotManager(options);
        var snapshotGraph = await snapshotManager.GetLastSavedDependencyGraphAsync(options, ct);

        var changedModules = await ChangeDetector.GetChangedProjectFilesAsync(options, snapshotGraph, ct);

        var parser = DependencyParserFactory.SelectDependencyParser(options);
        var graph = await new DependencyGraphBuilder(parser, options).GetGraphAsync(changedModules, ct);

        var renderer = RendererFactory.SelectRenderer(options.Format);
        var artifactPath = renderer.RenderGraph(graph, options, ct);

        await snapshotManager.SaveGraphAsync(graph, options, ct);

        if (options.Format == Domain.Models.Enums.RenderFormat.Json)
            await File.WriteAllTextAsync($"{options.FullRootPath.Replace("src\\c-sharp\\", "")}/diagrams/graph-json.json", artifactPath, ct);
        else if (options.Format == Domain.Models.Enums.RenderFormat.PlantUML)
            await File.WriteAllTextAsync($"{options.FullRootPath.Replace("src\\c-sharp\\", "")}/diagrams/graph-puml.puml", artifactPath, ct);

        return artifactPath;
    }
}