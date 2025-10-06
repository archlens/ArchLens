using System;
using System.Threading;
using System.Threading.Tasks;
using SyntaxTreeManualTraversal.Domain.Interfaces;
using SyntaxTreeManualTraversal.Domain.Models;
using SyntaxTreeManualTraversal.Domain.Models.Records;

namespace SyntaxTreeManualTraversal.Infra;

public sealed class GitBaselineManager : IBaselineManager
{
    public Task<string> GetLastSavedDependencyGraphAsync(Options options, CancellationToken ct = default)
    {
        throw new NotImplementedException();
    }

    public Task SaveGraphAsync(DependencyGraph graph, Options options, CancellationToken ct = default)
    {
        throw new NotImplementedException();
    }
}