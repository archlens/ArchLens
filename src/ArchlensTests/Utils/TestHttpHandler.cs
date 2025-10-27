using System.Collections.Concurrent;
using System.Net;

namespace ArchlensTests.Utils;

public sealed class TestHttpHandler : HttpMessageHandler
{
    private readonly ConcurrentDictionary<string, HttpResponseMessage> _map = new(StringComparer.OrdinalIgnoreCase);

    public void When(string url, HttpStatusCode status, string? body = null, string mediaType = "application/json")
    {
        var msg = new HttpResponseMessage(status)
        {
            Content = body is null ? null : new StringContent(body, System.Text.Encoding.UTF8, mediaType)
        };
        _map[url] = msg;
    }

    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        if (_map.TryGetValue(request.RequestUri!.ToString(), out var msg))
            return Task.FromResult(msg.Clone());

        return Task.FromResult(new HttpResponseMessage(HttpStatusCode.NotFound));
    }
}


internal static class HttpResponseMessageExtensions
{
    public static HttpResponseMessage Clone(this HttpResponseMessage msg)
    {
        var clone = new HttpResponseMessage(msg.StatusCode);
        if (msg.Content is not null)
        {
            var content = msg.Content.ReadAsStringAsync().GetAwaiter().GetResult();
            clone.Content = new StringContent(content, System.Text.Encoding.UTF8, msg.Content.Headers.ContentType?.MediaType ?? "application/json");
        }
        foreach (var h in msg.Headers) clone.Headers.TryAddWithoutValidation(h.Key, h.Value);
        clone.RequestMessage = new HttpRequestMessage();
        return clone;
    }
}
