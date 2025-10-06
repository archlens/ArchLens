using System.Threading;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Domain.Interfaces;

public interface IRenderer
{
    public string RenderGraph(DependencyGraph graph, Options options, CancellationToken ct = default);
}