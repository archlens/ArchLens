using System.Threading;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Infra;

public sealed class JsonRenderer : IRenderer
{
    public string RenderGraph(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        var packagestr = "";
        for (int i = 0; i < graph.packages().Count; i++)
        {
            var package = graph.packages()[i];

            if (packagestr.Contains(package)) continue;

            var comma = "";
            if (i < graph.packages().Count - 2) comma = ",";

            packagestr +=
                $$"""
                
                {
                    "name": "{{package}}",
                    "state": "NEUTRAL"
                }{{comma}}

            """;
        }

        var str =
        $$"""
        {
            "title": "{{graph.Name}}",
            "packages": [
                {{packagestr}}
            ],

            "edges": [
            {{graph.ToJson()}}

            ]
        }
        """;
        return str;
    }

}