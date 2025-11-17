using Archlens.Domain;
using Archlens.Domain.Models;
using Archlens.Domain.Models.Records;
using ArchlensTests.Utils;

namespace ArchlensTests.Domain;

public sealed class ChangeDetectorTests : IDisposable
{
    private readonly TestFileSystem _fs = new();
    private Options MakeOptions(IReadOnlyList<string>? exclusions = null, IReadOnlyList<string>? extensions = null)
        => new(
            ProjectRoot: _fs.Root,
            ProjectName: "TestProject",
            Language: default,
            SnapshotManager: default,
            Format: default,
            Exclusions: exclusions ?? [],
            FileExtensions: extensions ?? [".cs"],
            FullRootPath: _fs.Root
        );

    public void Dispose() => _fs.Dispose();

    private static SnapshotGraph MakeDefaultSnapshotGraph(string projectRoot)
    {
        return new SnapshotGraph(projectRoot)
        {
            Name = "src",
            Path = "./",
            LastWriteTime = DateTime.UtcNow
        };
    }

    private sealed class SnapshotGraph(string projectRoot) : DependencyGraph(projectRoot)
    {
        private readonly Dictionary<string, DependencyGraph> _nodes = new(StringComparer.OrdinalIgnoreCase);

        public void AddFile(string relPath, DateTime lastWriteUtc)
        {
            var n = new DependencyGraphLeaf(projectRoot) { Name = System.IO.Path.GetFileName(relPath), Path = "./" + relPath.Replace('\\', '/'), LastWriteTime = lastWriteUtc };
            var dir = System.IO.Path.GetDirectoryName(relPath)?.Replace('\\', '/') ?? ".";
            _nodes[relPath.Replace('\\', '/')] = n;
            _nodes[dir] = new DependencyGraphNode(projectRoot) { Name = dir, Path = "./" + dir, LastWriteTime = lastWriteUtc };
        }

        public override DependencyGraph GetChild(string path)
        {
            var key = path.Replace('\\', '/');
            if (_nodes.TryGetValue(key, out var n)) return n;
            if (_nodes.TryGetValue(key.TrimEnd('/'), out n)) return n;
            return null!;
        }
    }

    [Fact]
    public async Task Returns_NewFiles_When_NotInLastSavedGraph()
    {
        var t = DateTime.UtcNow;
        _fs.File("src/A.cs", "class A {}", t);

        var opts = MakeOptions();
        var snap = MakeDefaultSnapshotGraph(_fs.Root);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, snap);

        Assert.Contains(Path.Combine(_fs.Root, "src"), changed.Keys);
        Assert.Contains(Path.Combine(_fs.Root, "src", "A.cs"), changed[Path.Combine(_fs.Root, "src")]);
    }

    [Fact]
    public async Task DoesNotReturn_Unchanged_When_TimestampsEqual()
    {
        var t = DateTime.UtcNow.AddMinutes(-5);
        _fs.File("src/B.cs", "class B {}", t);

        var opts = MakeOptions();
        var snap = MakeDefaultSnapshotGraph(_fs.Root);
        snap.AddFile("src/B.cs", t);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, snap);

        Assert.Empty(changed);
    }

    [Fact]
    public async Task Returns_Modified_When_CurrentIsNewer()
    {
        var oldT = DateTime.UtcNow.AddMinutes(-10);
        var newT = DateTime.UtcNow.AddMinutes(-1);

        _fs.File("src/C.cs", "class C {}", newT);

        var opts = MakeOptions();
        var snap = MakeDefaultSnapshotGraph(_fs.Root);
        snap.AddFile("src/C.cs", oldT);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, snap);

        Assert.Single(changed);
        var mod = changed.Single();
        Assert.Equal(Path.Combine(_fs.Root, "src"), mod.Key);
        Assert.Contains(Path.Combine(_fs.Root, "src", "C.cs"), mod.Value);
    }

    [Fact]
    public async Task Respects_FileExtensions_Filter()
    {
        _fs.File("src/A.txt", "text");
        _fs.File("src/B.cs", "class B {}");

        var opts = MakeOptions(extensions: [".cs"]);
        var snap = MakeDefaultSnapshotGraph(_fs.Root);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, snap);

        var srcKey = Path.Combine(_fs.Root, "src");
        Assert.Contains(srcKey, changed.Keys);
        Assert.DoesNotContain(Path.Combine(_fs.Root, "src", "A.txt"), changed[srcKey]);
        Assert.Contains(Path.Combine(_fs.Root, "src", "B.cs"), changed[srcKey]);
    }

    [Fact]
    public async Task Excludes_DirectoryPrefix_RelativeWithSlash()
    {
        _fs.File("Tests/X.cs", "class X {}");
        _fs.File("src/Y.cs", "class Y {}");

        var opts = MakeOptions(exclusions: ["Tests/"]);
        var snap = MakeDefaultSnapshotGraph(_fs.Root);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, snap);

        Assert.DoesNotContain(Path.Combine(_fs.Root, "Tests"), changed.Keys);
        Assert.Contains(Path.Combine(_fs.Root, "src"), changed.Keys);
    }

    [Fact]
    public async Task Excludes_Segment_bin_Anywhere()
    {
        _fs.File("src/bin/Gen.cs", "class Gen {}");
        _fs.File("src/good/Ok.cs", "class Ok {}");

        var opts = MakeOptions(exclusions: ["bin"]);
        var snap = MakeDefaultSnapshotGraph(_fs.Root);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, snap);

        Assert.Contains(Path.Combine(_fs.Root, "src", "good"), changed.Keys);
        Assert.DoesNotContain(Path.Combine(_fs.Root, "src", "bin"), changed.Keys);
        Assert.DoesNotContain(Path.Combine(_fs.Root, "src", "bin", "Gen.cs"),
                              changed.GetValueOrDefault(Path.Combine(_fs.Root, "src")) ?? Array.Empty<string>());
    }

    [Fact]
    public async Task Excludes_FilenameSuffix_Wildcard_With_TrailingDot()
    {
        _fs.File("src/A.dev.cs", "class ADev {}");
        _fs.File("src/A.cs", "class A {}");

        var opts = MakeOptions(exclusions: ["**.dev.cs."]);
        var snap = MakeDefaultSnapshotGraph(_fs.Root);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, snap);

        var srcKey = Path.Combine(_fs.Root, "src");
        Assert.Contains(srcKey, changed.Keys);
        Assert.DoesNotContain(Path.Combine(_fs.Root, "src", "A.dev.cs"), changed[srcKey]);
        Assert.Contains(Path.Combine(_fs.Root, "src", "A.cs"), changed[srcKey]);
    }

    [Fact]
    public async Task Cancellation_Propagates()
    {
        using var cts = new CancellationTokenSource();
        _fs.File("src/A.cs", "class A {}");

        var opts = MakeOptions();
        var snap = MakeDefaultSnapshotGraph(_fs.Root);

        cts.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(async () =>
            await ChangeDetector.GetChangedProjectFilesAsync(opts, snap, cts.Token));
    }
}
