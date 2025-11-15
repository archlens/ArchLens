using System.IO;

namespace Archlens.Domain.Utils;

public static class PathNormaliser
{
    public static string NormalisePath(string root, string path)
    {
        var rootPath = Path.GetFullPath(root);
        var isModule = IsDirectoryPath(path);
        var relativePath = Path.GetRelativePath(rootPath, path).Replace(Path.DirectorySeparatorChar, '/');
        if (isModule)
            return  $"./{relativePath}/";
        return $"./{relativePath}";
    }

    public static bool IsDirectoryPath(string path)
    {
        bool isDir = false;
        try
        {
            isDir = (File.GetAttributes(path) & FileAttributes.Directory) != 0;
        }
        catch { /* ignore */ }
        return isDir;
    }
}
