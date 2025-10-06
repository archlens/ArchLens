using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;

namespace Archlens.Infra;

class CsharpDependencyParser : IDependencyParser
{
    public Task<IReadOnlyList<string>> ParseFileDependencies(string path, CancellationToken ct = default)
    {
        throw new System.NotImplementedException();
    }

    public Task<IReadOnlyList<string>> ParseModuleDependencies(string path, CancellationToken ct = default)
    {
        throw new System.NotImplementedException();
    }
}