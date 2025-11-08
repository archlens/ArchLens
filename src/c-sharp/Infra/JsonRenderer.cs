using System.Linq;
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
        var children = graph.GetChildren();
        for (int i = 0; i < children.Count; i++)
        {
            var package = children[i];

            if (packagestr.Contains(package.Name)) continue;

            if (i > 0) packagestr += ",\n";

            packagestr +=
                $$"""
                
                {
                    "name": "{{package.Name}}",
                    "state": "NEUTRAL"
                }
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