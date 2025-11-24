using Archlens.Domain.Interfaces;

namespace ArchlensTests.Utils;

public sealed class TestDependencyParser(IReadOnlyDictionary<string, IReadOnlyList<string>> map) : IDependencyParser
{
    private readonly IReadOnlyDictionary<string, IReadOnlyList<string>> _map = map;

    public Task<IReadOnlyList<string>> ParseFileDependencies(string path, CancellationToken ct = default)
    {
        _map.TryGetValue(path, out var deps);
        return Task.FromResult(deps ?? []);
    }
}
