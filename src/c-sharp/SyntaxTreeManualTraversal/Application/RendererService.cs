using System;
using System.Threading;
using System.Threading.Tasks;
using SyntaxTreeManualTraversal.Domain.Interfaces;
using SyntaxTreeManualTraversal.Domain;
using SyntaxTreeManualTraversal.Domain.Models.Enums;

namespace SyntaxTreeManualTraversal.Application;

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

        var changedFiles = await _changes.GetChangedProjectFilesAsync(options, baselineGraph, ct);

        var parser = _parserFactory(options.Language);
        var graph = await new DependencyGraphBuilder(parser).BuildGraphAsync(changedFiles, ct);

        var renderer = _rendererFactory(options.Format);
        var artifactPath = await renderer.RenderGraphAsync(graph, options, ct);

        await baselineManager.SaveGraphAsync(graph, options, ct); // Maybe not necessary

        return artifactPath;
    }
}