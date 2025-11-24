using Archlens.Domain;
using Archlens.Domain.Strategies;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Application;

public sealed class RendererService(string _configPath)
{
    public async Task RenderDependencyGraphAsync(CancellationToken ct = default)
    {
        var configManager = new ConfigManager(_configPath);

        var options = await configManager.LoadAsync(ct);

        var snapshotManager = SnapsnotManagerStrategy.SelectSnapshotManager(options);
        var snapshotGraph = await snapshotManager.GetLastSavedDependencyGraphAsync(options, ct);

        var changedModules = await ChangeDetector.GetChangedProjectFilesAsync(options, snapshotGraph, ct);

        var parser = DependencyParserStrategy.SelectDependencyParser(options);
        var graph = await new DependencyGraphBuilder(parser, options).GetGraphAsync(changedModules, ct);

        var renderer = RendererStrategy.SelectRenderer(options.Format);
        await renderer.SaveGraphToFileAsync(graph, options, ct);
        await snapshotManager.SaveGraphAsync(graph, options, ct);
    }
}