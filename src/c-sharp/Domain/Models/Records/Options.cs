using System.Collections.Generic;
using Archlens.Domain.Models.Enums;
namespace Archlens.Domain.Models.Records;

public sealed record Options(
    string ProjectRoot,
    string ProjectName,
    Language Language,
    SnapshotManager SnapshotManager,
    RenderFormat Format,
    IReadOnlyList<string> Exclusions,
    IReadOnlyList<string> FileExtensions,
    string SnapshotDir = ".archlens",
    string SnapshotFile = "snaphot",
    string GitUrl = "",
    string FullRootPath = ""
);