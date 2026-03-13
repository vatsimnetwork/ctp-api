FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS base
WORKDIR /app
EXPOSE 8080

FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src
COPY ["ctp-api/ctp-api.csproj", "ctp-api/"]
RUN dotnet restore "ctp-api/ctp-api.csproj"
COPY . .
WORKDIR "/src/ctp-api"
RUN dotnet build "ctp-api.csproj" -c Release -o /app/build

FROM build AS publish
RUN dotnet publish "ctp-api.csproj" -c Release -o /app/publish /p:UseAppHost=false

FROM base AS final
WORKDIR /app
COPY --from=publish /app/publish .
ENTRYPOINT ["dotnet", "ctp-api.dll"]
