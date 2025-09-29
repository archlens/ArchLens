using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;
using System.Collections.Generic;

namespace SyntaxTreeManualTraversal
{
    class UsingCollector : CSharpSyntaxWalker
    {
        public ICollection<UsingDirectiveSyntax> Usings { get; } = new List<UsingDirectiveSyntax>();
        private string _programName;

        public UsingCollector(string programName)
        {
            _programName = programName;
        }
        public override void VisitUsingDirective(UsingDirectiveSyntax node)
        {
            if (node.Name.ToString().StartsWith(_programName))
            {
                this.Usings.Add(node);
            }
        }
    }
}
