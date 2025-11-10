namespace Archlens.Domain.Factories;

using System;
using Archlens.Domain.Interfaces;
using Archlens.Domain.Models.Enums;
using Archlens.Infra;

public sealed class RendererFactory
{
    public static IRenderer SelectRenderer(RenderFormat f) => f switch
    {
        RenderFormat.Json => new JsonRenderer(),
        RenderFormat.PlantUML => new PlantUMLRenderer(),
        _ => throw new ArgumentOutOfRangeException(nameof(f))
    };
}