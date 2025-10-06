using System.Threading;
using System.Threading.Tasks;
using SyntaxTreeManualTraversal.Domain.Models;
using SyntaxTreeManualTraversal.Domain.Models.Records;

namespace SyntaxTreeManualTraversal.Domain.Interfaces;

public interface IRenderer
{
    public Task<string> RenderGraphAsync(DependencyGraph graph, Options options, CancellationToken ct = default);
}