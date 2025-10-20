using System;
using System.Collections.Generic;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Records;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;

namespace Archlens.Infra;

class CsharpSyntaxWalkerParser(Options _options) : CSharpSyntaxWalker, IDependencyParser
{
    public ICollection<UsingDirectiveSyntax> Usings { get; } = new List<UsingDirectiveSyntax>();

    public override void VisitUsingDirective(UsingDirectiveSyntax node)
    {
        if (node.Name.ToString().StartsWith(_options.ProjectName))
        {
            Usings.Add(node);
        }
    }

    public async Task<IReadOnlyList<string>> ParseFileDependencies(string path, CancellationToken ct = default)
    {

        string lines = "";
        List<string> usings = [];

        try
        {
            StreamReader sr = new(path);

            string line = await sr.ReadLineAsync();

            while (line != null)
            {
                lines += "\n" + line;
                line = await sr.ReadLineAsync();
            }

            sr.Close();
        }
        catch (Exception e)
        {
            Console.WriteLine("Exception: " + e.Message);
        }

        SyntaxTree tree = CSharpSyntaxTree.ParseText(lines, cancellationToken: ct);
        CompilationUnitSyntax root = tree.GetCompilationUnitRoot(ct);

        Visit(root);

        foreach (var directive in Usings)
        {
            usings.Add(directive.Name.ToString());
        }

        return usings;
    }

    public Task<IReadOnlyList<string>> ParseModuleDependencies(string path, CancellationToken ct = default)
    {
        throw new NotImplementedException();
    }
}