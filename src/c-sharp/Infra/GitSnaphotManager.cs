using Archlens.Domain;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using System;
using System.Net;
using System.Net.Http;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

namespace Archlens.Infra;

public sealed class GitSnaphotManager : ISnapshotManager
{
    private readonly string _gitDirName;
    private readonly string _gitFileName;
    private readonly LocalSnaphotManager _localManager;
    private readonly HttpClient _http; // injected for tests

    public GitSnaphotManager(string gitDirName, string gitFileName)
        : this(gitDirName, gitFileName, handler: null) { }

    public GitSnaphotManager(string gitDirName, string gitFileName, HttpMessageHandler? handler)
    {
        _gitDirName = gitDirName;
        _gitFileName = gitFileName;
        _localManager = new LocalSnaphotManager(gitDirName, gitFileName);
        _http = handler is null ? new HttpClient() : new HttpClient(handler, disposeHandler: true);
        _http.DefaultRequestHeaders.UserAgent.ParseAdd("Archlens-GitSnapshot/1.0");
    }

    public async Task SaveGraphAsync(DependencyGraph graph, Options options, CancellationToken ct = default)
        => await _localManager.SaveGraphAsync(graph, options, ct);
    
    public async Task<DependencyGraph> GetLastSavedDependencyGraphAsync(Options options, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(options.GitUrl))
            throw new ArgumentException("GitUrl must be provided for GitSnaphotManager. Options has registered GitUrl as Null or Whitespace - has it been correctly configured in .archlens json?");

        if (!TryParseGitHubRepo(options.GitUrl, out var owner, out var repo))
            throw new ArgumentException("Colud not parse GitUrl (accepted formats: https://github.com/owner/repo, https://github.com/owner/repo.git, http(s)://github.enterprise.tld/owner/repo).");

        var branches = new[] { "main", "master" };

        foreach (var branch in branches)
        {
            var url = BuildRawUrl(owner, repo, branch, _gitDirName, _gitFileName);

            try
            {
                var json = await HttpGetAsync(url, ct).ConfigureAwait(false);
                if (string.IsNullOrWhiteSpace(json))
                    continue;

                var graph = DependencyGraphSerializer.Deserialize(json);
                if (graph is not null) return graph;
            }
            catch (OperationCanceledException) { throw; }
            catch
            { /* ignore */ }
        }
        throw new Exception("Unable to find main branch's graph snapshot");
    }

    private static bool TryParseGitHubRepo(string url, out string owner, out string repo)
    {
        owner = repo = string.Empty;

        try
        {
            // Accept:
            // - https://github.com/owner/repo
            // - https://github.com/owner/repo.git
            // - http(s)://github.enterprise.tld/owner/repo
            var uri = new Uri(url);
            if (!uri.Host.EndsWith("github.com", StringComparison.OrdinalIgnoreCase))
                return false;

            var parts = uri.AbsolutePath.Trim('/').Split('/', StringSplitOptions.RemoveEmptyEntries);
            if (parts.Length != 2) return false;

            owner = parts[0];
            repo = parts[1];
            if (repo.EndsWith(".git", StringComparison.OrdinalIgnoreCase))
                repo = repo[..^4];

            return !string.IsNullOrWhiteSpace(owner) && !string.IsNullOrWhiteSpace(repo);
        }
        catch
        {
            return false;
        }
    }

    private static string BuildRawUrl(string owner, string repo, string branch, string dir, string file)
    {
        // https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{dir}/{file}
        static string Seg(string s) => WebUtility.UrlEncode(s ?? string.Empty).Replace("+", "%20");

        var sb = new StringBuilder("https://raw.githubusercontent.com/");
        sb.Append(Seg(owner)).Append('/').Append(Seg(repo)).Append('/').Append(Seg(branch)).Append('/');
        if (!string.IsNullOrWhiteSpace(dir))
        {
            var trimmed = dir.Trim('/', '\\');
            if (!string.IsNullOrEmpty(trimmed))
                sb.Append(Seg(trimmed)).Append('/');
        }
        sb.Append(Seg(file));
        return sb.ToString();
    }

    private async Task<string> HttpGetAsync(string url, CancellationToken ct)
    {
        using var resp = await _http.GetAsync(url, ct).ConfigureAwait(false);
        if (!resp.IsSuccessStatusCode) return null;
        return await resp.Content.ReadAsStringAsync(ct).ConfigureAwait(false);
    }
}