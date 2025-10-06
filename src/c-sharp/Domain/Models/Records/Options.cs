using System.Collections.Generic;
using Archlens.Domain.Models.Enums;
namespace Archlens.Domain.Models.Records;

public sealed record Options(
    string ProjectRoot,
    Language Language,
    Baseline Baseline,
    RenderFormat Format,
    IReadOnlyList<string> Exclusions,
    IReadOnlyList<string> FileExtensions
);