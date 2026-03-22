using System.Net;

namespace ctp_api.Middleware;

public class ApiKeyMiddleware
{
    private readonly RequestDelegate _next;
    private readonly IHttpClientFactory _httpClientFactory;

    public ApiKeyMiddleware(RequestDelegate next, IHttpClientFactory httpClientFactory)
    {
        _next = next;
        _httpClientFactory = httpClientFactory;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        if (!context.Request.Headers.TryGetValue("X-API-Key", out var extractedApiKey))
        {
            context.Response.StatusCode = (int)HttpStatusCode.Unauthorized;
            await context.Response.WriteAsync("API Key was not provided.");
            return;
        }   

        var client = _httpClientFactory.CreateClient("AuthService");
        var request = new HttpRequestMessage(HttpMethod.Get, "/internal/apikey/validate");
        request.Headers.Add("X-API-Key", extractedApiKey.ToString());
        var response = await client.SendAsync(request);

        if (!response.IsSuccessStatusCode)
        {
            context.Response.StatusCode = (int)HttpStatusCode.Unauthorized;
            await context.Response.WriteAsync("Unauthorized client.");
            return;
        }
        await _next(context);
    }

}

public static class ApiKeyMiddlewareExtensions
{
    public static IApplicationBuilder UseApiKeyMiddleware(this IApplicationBuilder builder)
    {
        return builder.UseMiddleware<ApiKeyMiddleware>();
    }
}