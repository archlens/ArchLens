using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Records;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Application;

public sealed class RendererService(Options options,
    ISnapshotManager snapshotManager,
    IDependencyParser parser,
    IRenderer renderer
    )
{
    public async Task RenderDependencyGraphAsync(CancellationToken ct = default)
    {
        var snapshotGraph = await snapshotManager.GetLastSavedDependencyGraphAsync(options, ct);
        var changedModules = await ChangeDetector.GetChangedProjectFilesAsync(options, snapshotGraph, ct);
        var graph = await new DependencyGraphBuilder(parser, options).GetGraphAsync(changedModules, ct);

        await renderer.SaveGraphToFileAsync(graph, options, ct);
        await snapshotManager.SaveGraphAsync(graph, options, ct);
    }
}