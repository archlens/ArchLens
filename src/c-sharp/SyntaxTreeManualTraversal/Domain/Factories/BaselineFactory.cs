namespace SyntaxTreeManualTraversal.Domain.Factories;

using System;
using SyntaxTreeManualTraversal.Domain.Interfaces;
using SyntaxTreeManualTraversal.Domain.Models.Enums;
using SyntaxTreeManualTraversal.Infra;

public sealed class BaselineFactory
{
    public static IBaselineManager SelectBaseline(Baseline b) => b switch
    {
        Baseline.Git   => new GitBaselineManager(),
        Baseline.Local => new LocalBaselineManager(),
        _ => throw new ArgumentOutOfRangeException(nameof(b))
    };
}