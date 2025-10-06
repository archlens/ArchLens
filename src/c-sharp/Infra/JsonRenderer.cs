using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Infra;

public sealed class JsonRenderer : IRenderer
{
    public Task<string> RenderGraphAsync(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        throw new System.NotImplementedException();
    }
}