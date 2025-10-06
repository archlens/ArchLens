using System.Threading;
using System.Threading.Tasks;
using SyntaxTreeManualTraversal.Domain.Interfaces;
using SyntaxTreeManualTraversal.Domain.Models;
using SyntaxTreeManualTraversal.Domain.Models.Records;

namespace SyntaxTreeManualTraversal.Infra;

public sealed class JsonRenderer : IRenderer
{
    public Task<string> RenderGraphAsync(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        throw new System.NotImplementedException();
    }
}