namespace SyntaxTreeManualTraversal.Domain.Factories;

using System;
using SyntaxTreeManualTraversal.Domain.Interfaces;
using SyntaxTreeManualTraversal.Domain.Models.Enums;
using SyntaxTreeManualTraversal.Infra;

public sealed class RendererFactory
{
    public static IRenderer SelectRenderer(RenderFormat f) => f switch
    {
        RenderFormat.Json     => new JsonRenderer(),
        _ => throw new ArgumentOutOfRangeException(nameof(f))
    };
}