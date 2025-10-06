using System.Collections.Generic;
using SyntaxTreeManualTraversal.Domain.Models.Enums;
namespace SyntaxTreeManualTraversal.Domain.Models.Records;

public sealed record Options(
    string ProjectRoot,
    Language Language,
    Baseline Baseline,
    RenderFormat Format,
    IReadOnlyList<string> Exclusions,
    IReadOnlyList<string> FileExtensions
);