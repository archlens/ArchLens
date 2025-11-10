using System;
using System.Collections.Generic;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Infra;

public sealed class PlantUMLRenderer : IRenderer
{
    public string RenderGraph(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        string title = options.ProjectName;
        List<string> graphString = graph.ToPlantUML(false); //TODO: diff
        graphString.Sort((s1, s2) => s1.Contains("package") ? (s2.Contains("package") ? 0 : -1) : (s2.Contains("package") ? 1 : 0));

        string uml_str = $"""
        @startuml
        skinparam linetype ortho
        skinparam backgroundColor GhostWhite
        title {title}
        {string.Join("\n", graphString.ToArray())}
        @enduml
        """;

        return uml_str;
    }

    public async Task SaveGraphToFileAsync(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        var filename = options.FullRootPath.Replace("src\\c-sharp\\", "") + "/diagrams/graph-puml.puml"; //TODO
        var content = RenderGraph(graph, options, ct);
        await File.WriteAllTextAsync(filename, content, ct);
    }
}