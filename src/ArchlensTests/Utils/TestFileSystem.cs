namespace ArchlensTests.Utils;

public class TestFileSystem : IDisposable
{
    public string Root { get; }

    public TestFileSystem()
    {
        Root = Path.Combine(Path.GetTempPath(), "archlens-tests", Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(Root);
    }

    public string Dir(params string[] segments)
    {
        var p = Path.Combine(new[] { Root }.Concat(segments).ToArray());
        Directory.CreateDirectory(p);
        return p;
    }

    public string File(string relPath, string contents = "", DateTime? lastWriteUtc = null)
    {
        var abs = Path.Combine(Root, relPath.Replace('/', Path.DirectorySeparatorChar));
        Directory.CreateDirectory(Path.GetDirectoryName(abs)!);
        System.IO.File.WriteAllText(abs, contents);
        if (lastWriteUtc is { } t)
        {
            Directory.SetLastWriteTimeUtc(Path.GetDirectoryName(abs)!, t);
            System.IO.File.SetLastWriteTimeUtc(abs, t);
        }
        return abs;
    }

    public void Dispose()
    {
        try { Directory.Delete(Root, recursive: true); } catch { /* ignore */ }
    }
}
