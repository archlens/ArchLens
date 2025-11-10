using System;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain;
using Archlens.Domain.Factories;

namespace Archlens.Application;

public sealed class RendererService(ConfigManager _config)
{

    public async Task RenderDependencyGraphAsync(CancellationToken ct = default)
    {
        var options = await _config.LoadAsync(ct);

        var snapshotManager = SnapsnotManagerFactory.SelectSnapshotManager(options);
        var snapshotGraph = await snapshotManager.GetLastSavedDependencyGraphAsync(options, ct);

        var changedModules = await ChangeDetector.GetChangedProjectFilesAsync(options, snapshotGraph, ct);

        var parser = DependencyParserFactory.SelectDependencyParser(options);
        var graph = await new DependencyGraphBuilder(parser, options).GetGraphAsync(changedModules, ct);

        var renderer = RendererFactory.SelectRenderer(options.Format);
        await renderer.SaveGraphToFileAsync(graph, options, ct);
        await snapshotManager.SaveGraphAsync(graph, options, ct);
    }
}