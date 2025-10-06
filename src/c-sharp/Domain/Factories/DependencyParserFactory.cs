namespace Archlens.Domain.Factories;

using System;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Enums;
using Archlens.Infra;

public sealed class DependencyParserFactory
{
    public static IDependencyParser SelectDependencyParser(Language l) => l switch
    {
        Language.CSharp => new CsharpDependencyParser(),
        _ => throw new ArgumentOutOfRangeException(nameof(l))
    };
}