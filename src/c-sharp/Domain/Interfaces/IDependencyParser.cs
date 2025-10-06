using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Domain.Interfaces;

public interface IDependencyParser
{
    Task<IReadOnlyList<string>> ParseModuleDependencies(string path, CancellationToken ct = default);
    Task<IReadOnlyList<string>> ParseFileDependencies(string path, CancellationToken ct = default);
}