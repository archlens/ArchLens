using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
namespace Archlens.Domain.Interfaces;

public interface ISnapshotManager
{
    Task SaveGraphAsync(DependencyGraph graph,
                   Options options,
                   CancellationToken ct = default);
    Task<DependencyGraph> GetLastSavedDependencyGraphAsync(Options options, CancellationToken ct = default);
}