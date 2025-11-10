using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Domain.Interfaces;

public interface IRenderer
{
    public string RenderGraph(DependencyGraph graph, Options options, CancellationToken ct = default);
    public Task SaveGraphToFileAsync(DependencyGraph graph, Options options, CancellationToken ct = default);
}