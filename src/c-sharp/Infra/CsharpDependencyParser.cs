using System;
using System.Collections.Generic;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;

namespace Archlens.Infra;

class CsharpDependencyParser(string _projectName) :  IDependencyParser
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
                line = await sr.ReadLineAsync(ct);
                // match regex + add to usings
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

    public Task<IReadOnlyList<string>> ParseModuleDependencies(string path, CancellationToken ct = default)
    {
        //for each file in module --> Parse file dependencies
        //for each module in module --> Parse module dependencies
        throw new System.NotImplementedException();
    }
}
