using System.Threading;
using System.Threading.Tasks;
using SyntaxTreeManualTraversal.Domain.Models;
using SyntaxTreeManualTraversal.Domain.Models.Records;
namespace SyntaxTreeManualTraversal.Domain.Interfaces;

public interface IBaselineManager
{
    Task SaveGraphAsync(DependencyGraph graph,
                   Options options,
                   CancellationToken ct = default);
    Task<string> GetLastSavedDependencyGraphAsync(Options options, CancellationToken ct = default);
}