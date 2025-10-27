using Archlens.Domain;
using Archlens.Domain.Models.Records;
using ArchlensTests.Utils.TestModels;

namespace ArchlensTests.Domain;
public sealed class ChangeDetectorTests : IDisposable
{
    private readonly string _root;

    public ChangeDetectorTests()
    {
        _root = Path.Combine(Path.GetTempPath(), "archlens-tests", Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(_root);
    }

    public void Dispose()
    {
        try { Directory.Delete(_root, recursive: true); } catch { /* ignore */ }
    }

    private static void WriteFile(string path, string contents, DateTime? utcWriteTime = null)
    {
        var dir = Path.GetDirectoryName(path);
        Directory.CreateDirectory(dir!);
        File.WriteAllText(path, contents);
        if (utcWriteTime is { } t)
        {
            Directory.SetLastWriteTimeUtc(dir!, t);
            File.SetLastWriteTimeUtc(path, t);
        }
    }

    private static Options MakeOptions(
        string projectRoot,
        IReadOnlyList<string>? exclusions = null,
        IReadOnlyList<string>? extensions = null
    )
    {
        return new Options(
            ProjectRoot: projectRoot,
            ProjectName: "TestProject",
            Language: default,
            SnapshotManager: default,
            Format: default,
            Exclusions: exclusions ?? [],
            FileExtensions: extensions ?? [".cs"]
        );
    }

    [Fact]
    public async Task Returns_NewFiles_When_NotInLastSavedGraph()
    {
        var now = DateTime.UtcNow;
        var path = Path.Combine(_root, "src", "A.cs");
        WriteFile(path, "class A {}", now);

        var opts = MakeOptions(_root);
        var depGraph = new TestDependencyGraph();

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, depGraph);

        Assert.Contains("src", changed);
        Assert.Contains("A.cs", changed["src"]);
        Assert.Single(changed);
    }

    [Fact]
    public async Task DoesNotReturn_UnchangedFiles_When_TimestampsEqual()
    {
        var t = DateTime.UtcNow.AddMinutes(-5);
        var path = Path.Combine(_root, "src", "B.cs");
        WriteFile(path, "class B {}", t);

        var opts = MakeOptions(_root);
        var depGraph = new TestDependencyGraph();
        depGraph.SetFile("src/B.cs", t);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, depGraph);

        Assert.Empty(changed);
    }

    [Fact]
    public async Task Returns_ModifiedFiles_When_CurrentIsNewer()
    {
        var oldT = DateTime.UtcNow.AddMinutes(-10);
        var newT = DateTime.UtcNow.AddMinutes(-1);

        var path = Path.Combine(_root, "src", "C.cs");
        WriteFile(path, "class C {}", newT);

        var opts = MakeOptions(_root);
        var depGraph = new TestDependencyGraph();
        depGraph.SetFile("src/C.cs", oldT);

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, depGraph);

        Assert.Contains("src", changed);
        Assert.Contains("C.cs", changed["src"]);
        Assert.Single(changed);
    }

    [Fact]
    public async Task Respects_FileExtensions_Filter()
    {
        var a = Path.Combine(_root, "src", "A.txt");
        var b = Path.Combine(_root, "src", "B.cs");
        WriteFile(a, "text");
        WriteFile(b, "class B {}");

        var opts = MakeOptions(_root, extensions: [".cs"]);
        var depGraph = new TestDependencyGraph();

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, depGraph);

        Assert.Contains("src", changed);
        Assert.Contains("B.cs", changed["src"]);
        Assert.DoesNotContain("A.txt", changed["src"]);
        Assert.Single(changed);
    }

    [Fact]
    public async Task Excludes_DirectoryPrefix_TestsSlash()
    {
        var t1 = Path.Combine(_root, "Tests", "X.cs");
        var t2 = Path.Combine(_root, "src", "Y.cs");
        WriteFile(t1, "class X {}");
        WriteFile(t2, "class Y {}");

        var opts = MakeOptions(_root, exclusions: ["**/Tests/"], extensions: [".cs"]);
        var depGraph = new TestDependencyGraph();

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, depGraph);

        Assert.Contains("src", changed);
        Assert.DoesNotContain("Tests", changed);
        Assert.Single(changed);
    }

    [Fact]
    public async Task Excludes_Segment_bin_AnywhereInPath()
    {
        var p1 = Path.Combine(_root, "src", "bin", "Gen.cs");
        var p2 = Path.Combine(_root, "src", "good", "Ok.cs");
        WriteFile(p1, "class Gen {}");
        WriteFile(p2, "class Ok {}");

        var opts = MakeOptions(_root, exclusions: ["bin"]);
        var depGraph = new TestDependencyGraph();

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, depGraph);

        Assert.Contains("src", changed);
        Assert.Contains("src/good", changed);
        Assert.DoesNotContain("bin", changed);
        Assert.DoesNotContain("bin", changed["src"]);
    }

    [Fact]
    public async Task Excludes_FilenameSuffix_dev_cs_With_TrailingDot()
    {
        var p1 = Path.Combine(_root, "src", "A.dev.cs");
        var p2 = Path.Combine(_root, "src", "A.cs");
        
        WriteFile(p1, "class ADev {}");
        WriteFile(p2, "class A {}");

        var opts = MakeOptions(_root, exclusions: ["**/*.dev.cs."]);
        var depGraph = new TestDependencyGraph();

        var changed = await ChangeDetector.GetChangedProjectFilesAsync(opts, depGraph);

        Assert.Contains("src", changed);
        Assert.DoesNotContain("A.dev.cs", changed["src"]);
        Assert.Single(changed);
    }
}
