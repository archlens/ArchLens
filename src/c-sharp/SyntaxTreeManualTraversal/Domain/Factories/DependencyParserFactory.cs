namespace SyntaxTreeManualTraversal.Domain.Factories;

using System;
using SyntaxTreeManualTraversal.Domain.Interfaces;
using SyntaxTreeManualTraversal.Domain.Models.Enums;
using SyntaxTreeManualTraversal.Infra;

public sealed class DependencyParserFactory
{
    public static IDependencyParser SelectDependencyParser(Language l) => l switch
    {
        Language.CSharp => new CsharpDependencyParser(),
        _ => throw new ArgumentOutOfRangeException(nameof(l))
    };
}