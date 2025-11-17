using System.IO;

namespace Archlens.Domain.Utils;

public static class PathNormaliser
{
    public static string NormalisePath(string root, string path)
    {
        if (!Path.IsPathFullyQualified(root))
            root = Path.GetFullPath(root);
        var fullPath = Path.IsPathRooted(path)
            ? path
            : Path.GetFullPath(path, root);

        var isDirectory = IsDirectoryPath(fullPath);

        var relativePath = Path
            .GetRelativePath(root, fullPath)
            .Replace(Path.DirectorySeparatorChar, '/')
            .TrimEnd('/');

        if (relativePath == "." || relativePath == string.Empty)
            return "./";

        return isDirectory ? $"./{relativePath}/" : $"./{relativePath}";
    }

    private static bool IsDirectoryPath(string fullPath)
    {
        try
        {
            return Directory.Exists(fullPath);
        }
        catch
        {
            return false;
        }
    }
}
