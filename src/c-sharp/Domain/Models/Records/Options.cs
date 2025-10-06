using System.Collections.Generic;
using Archlens.Domain.Models.Enums;
namespace Archlens.Domain.Models.Records;

public sealed record Options(
    string ProjectRoot,
    string ProjectName,
    Language Language,
    Baseline Baseline,
    RenderFormat Format,
    IReadOnlyList<string> Exclusions,
    IReadOnlyList<string> FileExtensions
);