using System;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain;
using Archlens.Domain.Models.Enums;

namespace Archlens.Application;

public sealed class RendererService(ConfigManager _config,
        ChangeDetector _changes,
        Func<Baseline, IBaselineManager> _baselineFactory,
        Func<Language, IDependencyParser> _parserFactory,
        Func<RenderFormat, IRenderer> _rendererFactory)
{

    public async Task<string> RenderDependencyGraphAsync(CancellationToken ct = default)
    {
        var options = await _config.LoadAsync(ct);

        var baselineManager = _baselineFactory(options.Baseline);
        var baselineGraph = await baselineManager.GetLastSavedDependencyGraphAsync(options, ct);

        var changedModules = await _changes.GetChangedProjectFilesAsync(options, baselineGraph, ct);

        var parser = _parserFactory(options.Language);
        var graph = await new DependencyGraphBuilder(parser).GetGraphAsync(options.ProjectRoot,changedModules, ct);

        var renderer = _rendererFactory(options.Format);
        var artifactPath = renderer.RenderGraph(graph, options, ct); // currently not being written to any file

        await baselineManager.SaveGraphAsync(graph, options, ct); // Maybe not necessary

        return artifactPath;
    }
}