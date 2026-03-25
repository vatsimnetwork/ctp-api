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

var authUri = builder.Configuration["AuthServiceURL"];
if (string.IsNullOrEmpty(authUri))
{
    throw new InvalidOperationException("AuthServiceURL is not configured. Please set it in the .env file.");
}
builder.Services.AddHttpClient("AuthService", client =>
{
    client.BaseAddress = new Uri(authUri);
    client.DefaultRequestHeaders.Add("X-API-Key", builder.Configuration["AuthServiceAPIKey"]);
});

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseApiKeyMiddleware();
app.UseHttpsRedirection();

app.MapControllers();

app.Run();