using ctp_api.Context;
using ctp_api.Middleware;
using DotNetEnv;
using Microsoft.EntityFrameworkCore;

var builder = WebApplication.CreateBuilder(args);

Env.Load();
builder.Configuration.AddEnvironmentVariables();

builder.Services.AddControllers();
builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(builder.Configuration.GetConnectionString("DefaultConnection")));

builder.Services.AddOpenApi();

var authUriRaw = builder.Configuration["AuthServiceURL"];
if (string.IsNullOrWhiteSpace(authUriRaw))
{
    throw new InvalidOperationException("AuthServiceURL is not configured. Please set it in the .env file.");
}

var authUriNormalized = authUriRaw.Trim();
if (!authUriNormalized.Contains("://", StringComparison.Ordinal))
{
    authUriNormalized = $"http://{authUriNormalized}";
}

if (!Uri.TryCreate(authUriNormalized, UriKind.Absolute, out var authBaseUri)
    || (authBaseUri.Scheme != Uri.UriSchemeHttp && authBaseUri.Scheme != Uri.UriSchemeHttps))
{
    throw new InvalidOperationException($"AuthServiceURL is invalid: '{authUriRaw}'. Provide a full URL like 'http://host.docker.internal:9000'.");
}

builder.Services.AddHttpClient("AuthService", client =>
{
    client.BaseAddress = authBaseUri;
    client.DefaultRequestHeaders.Add("X-API-Key", builder.Configuration["AuthServiceAPIKey"]);
});

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseApiKeyMiddleware();
if (!app.Environment.IsDevelopment())
{
    app.UseHttpsRedirection();
}

app.MapControllers();

app.Run();