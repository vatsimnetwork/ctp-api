# API Authentication

To authenticate with the API, you must include a valid API key corresponding to your service in the HTTP request headers. 

### Header Format
Set the `X-API-Key` header with your API key:
```http
X-API-Key: your_api_key_here
```

### Error Responses
- **`401 Unauthorized`**: Returned if the `X-API-Key` header is missing, or the provided API key is invalid.