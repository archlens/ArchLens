using System;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra.Parsers;

namespace Archlens.Infra.Factories;

public sealed class DependencyParserFactory
{
    public static IDependencyParser SelectDependencyParser(Options o) => o.Language switch
    {
        Language.CSharp => new CsharpDependencyParser(o),
        _ => throw new ArgumentOutOfRangeException(nameof(o.Language))
    };
}