using System;
using System.Collections.Generic;
using System.IO;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Records;

namespace Archlens.Infra;

class CsharpDependencyParser(Options _options) : IDependencyParser
{
    public async Task<IReadOnlyList<string>> ParseFileDependencies(string path, CancellationToken ct = default)
    {
        /*
            open file from given path
            match regex "Using.ProjectName"
            take all matches and put in list
            return list
        */
        List<string> usings = [];

        try
        {
            StreamReader sr = new(path);

            string line = await sr.ReadLineAsync(ct);

            while (line != null)
            {
                string regex = $$"""using\s+{{_options.ProjectName}}\.(.+);""";
                var match = Regex.Match(line, regex);
                if (match.Success)
                {
                    usings.Add(match.Groups[1].Value);
                }
                line = await sr.ReadLineAsync(ct);
            }

            sr.Close();
            return usings;
        }
        catch (Exception e)
        {
            Console.WriteLine("Exception: " + e.Message);
            return [];
        }

    }
}
