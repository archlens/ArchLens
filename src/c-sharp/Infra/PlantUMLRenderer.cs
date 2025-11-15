using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Infra;

public sealed class PlantUMLRenderer : IRenderer
{
    public string RenderGraph(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        string title = options.ProjectName;
        List<string> graphString = ToPlantUML(graph, false); //TODO: diff
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

    public static List<string> ToPlantUML(DependencyGraph graph, bool diff)
    {
        return graph switch
        {
            DependencyGraphNode node => NodeToPuml(node, diff),
            DependencyGraphLeaf leaf => LeafToPuml(leaf, diff),
            _ => throw new InvalidOperationException("Unknown DependencyGraph type"),
        };
    }

    private static List<string> NodeToPuml(DependencyGraphNode node, bool diff)
    {
        //TODO: Add color depending on diff
        string package = $"package \"{node.Name}\" as {node.Name} {{ \n";

        List<string> puml = [];

        foreach (var child in node.GetChildren())
        {
            string childName = child.Name.Replace(" ", "-");

            if (child is DependencyGraphLeaf)
            {
                package += $"\n [{childName}]";
                var childList = ToPlantUML(child, diff);
                puml.AddRange(childList);
            }
            else
            {
                var childList = ToPlantUML(child, diff);
                var c = childList.Last(); //last is the package declaration, which we want to be added here
                package += $"\n{c}\n";
                childList.Remove(c);
                puml.AddRange(childList);
            }
        }
        package += "\n}\n";
        puml.Add(package);
        return puml;
    }

    private static List<string> LeafToPuml(DependencyGraphLeaf leaf, bool diff)
    {
        //TODO: diff
        List<string> puml = [];

        foreach (var dep in leaf.GetDependencies().Keys)
        {
            puml.Add($"\n\"{leaf.Name}\"-->{dep}"); //package alias
        }
        return puml;
    }
}