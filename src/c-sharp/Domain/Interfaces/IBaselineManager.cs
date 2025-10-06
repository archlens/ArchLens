using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
namespace Archlens.Domain.Interfaces;

public interface IBaselineManager
{
    Task SaveGraphAsync(DependencyGraph graph,
                   Options options,
                   CancellationToken ct = default);
    Task<string> GetLastSavedDependencyGraphAsync(Options options, CancellationToken ct = default);
}