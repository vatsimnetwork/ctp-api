using ctp_api.Middleware;
using DotNetEnv;

var builder = WebApplication.CreateBuilder(args);

Env.Load();
builder.Configuration.AddEnvironmentVariables();
builder.Services.AddOpenApi();

var authUri = builder.Configuration["AuthServiceURL"];
if (string.IsNullOrEmpty(authUri))
{
    throw new InvalidOperationException("AuthServiceURL is not configured. Please set it in the .env file.");
}
builder.Services.AddHttpClient("AuthService", client =>
{
    client.BaseAddress = new Uri(authUri);
});

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}
app.UseApiKeyMiddleware();
app.UseHttpsRedirection();



app.Run();