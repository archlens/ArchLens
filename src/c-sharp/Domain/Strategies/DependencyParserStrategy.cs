using System;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra.Parsers;

namespace Archlens.Domain.Strategies;

public sealed class DependencyParserStrategy
{
    public static IDependencyParser SelectDependencyParser(Options o) => o.Language switch
    {
        Language.CSharp => new CsharpSyntaxWalkerParser(o),
        _ => throw new ArgumentOutOfRangeException(nameof(o.Language))
    };
}