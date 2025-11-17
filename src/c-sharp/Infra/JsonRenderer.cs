using System;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Infra;

public sealed class JsonRenderer : IRenderer
{
    public string RenderGraph(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        var childrenJson = "";
        var childrenRelations = "";
        var children = graph.GetChildren();
        for (int i = 0; i < children.Count; i++)
        {
            var child = children[i];

            if (childrenJson.Contains(child.Name)) continue;
            if (!childrenJson.EndsWith(",\n") && !string.IsNullOrEmpty(childrenJson)) childrenJson += ",\n";
            if (!childrenRelations.EndsWith(",\n") && !string.IsNullOrEmpty(childrenRelations) && !string.IsNullOrEmpty(child.ToJson())) childrenRelations += ",\n";

            childrenJson +=
                $$"""
                
                {
                    "name": "{{child.Name}}",
                    "state": "NEUTRAL"
                }
            """;

            childrenRelations += child.ToJson();
        }

        var str =
        $$"""
        {
            "title": "{{graph.Name}}",
            "packages": [
                {{childrenJson}}
            ],

            "edges": [
                {{childrenRelations}}
            ]
        }
        """;
        return str;
    }

    public async Task SaveGraphToFileAsync(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        var filename = options.FullRootPath.Replace("src\\c-sharp\\", "") + "/diagrams/graph-json.json"; //TODO
        var content = RenderGraph(graph, options, ct);
        await File.WriteAllTextAsync(filename, content, ct);
    }

}