using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;

namespace Archlens.Domain;

public class ChangeDetector(DependencyGraph _lastsavedState,
                    string _projectRoot,
                    string _exclusions,
                    IList<string> _languageExtensions)
{
    public Task<IReadOnlyList<string>> GetChangedProjectFilesAsync(Options options, string baselineGraph, CancellationToken ct = default)
    {
        throw new System.NotImplementedException();
    }
}