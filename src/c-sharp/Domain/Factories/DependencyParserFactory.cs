using System;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Enums;
using Archlens.Domain.Models.Records;
using Archlens.Infra;

namespace Archlens.Domain.Factories;
public sealed class DependencyParserFactory
{
    public static IDependencyParser SelectDependencyParser(Options o) => o.Language switch
    {
        Language.CSharp => new CsharpSyntaxWalkerParser(o),
        _ => throw new ArgumentOutOfRangeException(nameof(o.Language))
    };
}