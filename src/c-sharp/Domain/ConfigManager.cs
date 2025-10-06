using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Models.Records;

namespace Archlens.Domain;

public class ConfigManager
{
    public Task<Options> LoadAsync(CancellationToken ct = default)
    {
        throw new System.NotImplementedException();
    }
}