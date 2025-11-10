using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;

namespace Archlens.Domain;

public class ConfigManager(string _path)
{
    private sealed class ConfigDto
    {
#pragma warning disable CS8632
        public string? ProjectRoot { get; set; }
        public string? ProjectName { get; set; }
        public string? Language { get; set; }
        public string? SnapshotManager { get; set; }
        public string? Format { get; set; }
        public string[]? Exclusions { get; set; }
        public string[]? FileExtensions { get; set; }
#pragma warning restore CS8632
    }

    public async Task<Options> LoadAsync(CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(_path))
            throw new ArgumentException("Config path is null/empty.", nameof(_path));

        var configFile = Path.GetFullPath(_path);
        if (!File.Exists(configFile))
            throw new FileNotFoundException($"Config file not found: {configFile}", configFile);

        await using var fileStream = File.OpenRead(configFile);

        var dto = await JsonSerializer.DeserializeAsync<ConfigDto>(
            fileStream,
            new JsonSerializerOptions { PropertyNameCaseInsensitive = true },
            ct
        ) ?? throw new InvalidOperationException($"Could not parse JSON in {configFile}.");

        var baseDir = Path.GetDirectoryName(configFile) ?? Environment.CurrentDirectory;

        var options = MapOptions(dto, baseDir);

        return options;
    }

    private static Options MapOptions(ConfigDto dto, string baseDir)
    {
        var projectRoot = dto.ProjectRoot ?? baseDir;
        var projectName = dto.ProjectName ?? baseDir.Split("\\").Last();
        var language = MapLanguage(dto.Language ?? "c#");
        var snapshotManager = MapSnapshotManager(dto.SnapshotManager ?? "git");
        var format = MapFormat(dto.Format ?? "json");
        var exclusions = (dto.Exclusions ?? []).Select(s => s.Trim())
            .Where(s => !string.IsNullOrWhiteSpace(s))
            .ToArray();

        var fileExts = (dto.FileExtensions ?? DefaultExtensions(language)).Select(NormalizeExtension).ToArray();

        if (!Directory.Exists(projectRoot))
            throw new DirectoryNotFoundException($"projectRoot does not exist: {projectRoot}");

        var fullRootPath = GetFullRootPath(projectRoot);

        if (fileExts.Length == 0)
            throw new InvalidOperationException("fileExtensions resolved to an empty list.");

        return new Options(
            ProjectRoot: projectRoot,
            ProjectName: projectName,
            Language: language,
            SnapshotManager: snapshotManager,
            Format: format,
            Exclusions: fileExts.Length == 0 ? [] : exclusions,
            FileExtensions: fileExts,
            FullRootPath: fullRootPath
        );
    }

    private static string GetFullRootPath(string root)
    {
        return Path.GetFullPath(root);
    }

    private static string NormalizeExtension(string ext)
    {
        ext = ext.Trim();
        return ext.StartsWith('.') ? ext : "." + ext;
    }

    private static IReadOnlyList<string> DefaultExtensions(Language lang) => lang switch
    {
        Language.CSharp => [".cs"],
        _ => []
    };

    private static Language MapLanguage(string raw)
    {
        var s = raw.Trim().ToLowerInvariant();
        return s switch
        {
            "c#" or "csharp" or "cs" or "c-sharp" or "c sharp" => Language.CSharp,
            _ => throw new NotSupportedException($"Unsupported language: '{raw}'.")
        };
    }

    private static SnapshotManager MapSnapshotManager(string raw)
    {
        var s = raw.Trim().ToLowerInvariant();
        return s switch
        {
            "git" => SnapshotManager.Git,
            "local" => SnapshotManager.Local,
            _ => throw new NotSupportedException($"Unsupported baseline: '{raw}'.")
        };
    }

    private static RenderFormat MapFormat(string raw)
    {
        var s = raw.Trim().ToLowerInvariant();
        return s switch
        {
            "json" or "application/json" => RenderFormat.Json,
            "puml" or "plantuml" or "plant-uml" => RenderFormat.PlantUML,
            _ => throw new NotSupportedException($"Unsupported format: '{raw}'.")
        };
    }
}
