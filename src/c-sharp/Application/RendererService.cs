using System;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain;
using Archlens.Domain.Models.Enums;
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
        var graph = await new DependencyGraphBuilder(parser, options).GetGraphAsync(options.ProjectRoot, changedModules, ct);

        var renderer = RendererFactory.SelectRenderer(options.Format);
        var artifactPath = renderer.RenderGraph(graph, options, ct); // currently not being written to any file

        await snapshotManager.SaveGraphAsync(graph, options, ct); // Maybe not necessary

        return artifactPath;
    }
}