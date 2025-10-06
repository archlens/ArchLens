namespace Archlens.Domain.Factories;

using System;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Enums;
using Archlens.Infra;

public sealed class BaselineFactory
{
    public static IBaselineManager SelectBaseline(Baseline b) => b switch
    {
        Baseline.Git   => new GitBaselineManager(),
        Baseline.Local => new LocalBaselineManager(),
        _ => throw new ArgumentOutOfRangeException(nameof(b))
    };
}