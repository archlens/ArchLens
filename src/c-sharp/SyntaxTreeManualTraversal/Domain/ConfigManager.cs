using System.Threading;
using System.Threading.Tasks;
using SyntaxTreeManualTraversal.Domain.Models.Records;

namespace SyntaxTreeManualTraversal.Domain;

public class ConfigManager
{
    public Task<Options> LoadAsync(CancellationToken ct = default)
    {
        throw new System.NotImplementedException();
    }
}