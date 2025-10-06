using System.Collections.Generic;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;

namespace Archlens.Domain;

// For a single file, scan dependencies from this project
class UsingCollector(string programName) : CSharpSyntaxWalker
{
    public ICollection<UsingDirectiveSyntax> Usings { get; } = new List<UsingDirectiveSyntax>();
    private string _programName = programName;


    public override void VisitUsingDirective(UsingDirectiveSyntax node)
    {
        if (node.Name.ToString().StartsWith(_programName))
        {
            this.Usings.Add(node);
        }
    }
}
