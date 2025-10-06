namespace Archlens.Domain.Factories;

using System;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra;

public sealed class DependencyParserFactory
{
    public static IDependencyParser SelectDependencyParser(Language l, Options o) => l switch
    {
        Language.CSharp => new CsharpDependencyParser(o),
        _ => throw new ArgumentOutOfRangeException(nameof(l))
    };
}