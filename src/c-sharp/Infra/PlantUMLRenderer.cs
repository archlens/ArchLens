using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using System;
using System.Collections.Generic;
using System.IO;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Infra;

public sealed class PlantUMLRenderer : IRenderer
{
    public string RenderGraph(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        string title = options.ProjectName;
        List<string> graphString = ToPlantUML(graph); //TODO: diff
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

    public static List<string> ToPlantUML(DependencyGraph graph, bool isRoot = true)
    {
        return graph switch
        {
            DependencyGraphNode node => NodeToPuml(node, isRoot),
            DependencyGraphLeaf => [],
            _ => throw new InvalidOperationException("Unknown DependencyGraph type"),
        };
    }

    private static List<string> NodeToPuml(DependencyGraphNode node, bool isRoot = true)
    {
        List<string> puml = [];

        if (isRoot)
        {
            foreach (var child in node.GetChildren())
            {
                if (child is DependencyGraphNode)
                {
                    var childList = ToPlantUML(child, false);

                    puml.AddRange(childList);
                }
            }
        }
        else
        {
            puml.Add($"package \"{node.Name.Replace("\\", ".")}\" as {node.Name.Replace("\\", ".")} {{ }}");

            foreach (var (dep, count) in node.GetDependencies())
            {
                var fromName = node.Name;
                var toName = dep.Split(".")[0];
                var existing = puml.Find(p => p.StartsWith($"{fromName}-->{toName} : "));
                if (string.IsNullOrEmpty(existing))
                    puml.Add($"{fromName}-->{toName} : {count}"); //TODO: Add color depending on diff
                else
                {
                    var existingCount = existing.Replace($"{fromName}-->{toName} : ", "");
                    var canParse = int.TryParse(existingCount, out var exCount);

                    if (!canParse) Console.WriteLine("Error parsing " + existingCount);

                    var newCount = canParse ? exCount + count : count;

                    puml.Remove(existing);
                    puml.Add($"{fromName}-->{toName} : {newCount}"); //TODO: Add color depending on diff
                }
            }
        }
        return puml;
    }
}