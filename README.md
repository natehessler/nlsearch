# NLSearch - Natural Language Code Search

A web application that converts natural language queries into Sourcegraph code search queries using Deep Search API.

## Features

- 🔍 Natural language to search query translation
- 🌐 Clean, modern web interface
- ⚡ Real-time Deep Search API integration
- 🎯 Example queries to get started quickly
- 📊 Source attribution for search results

## Prerequisites

- Go 1.23 or higher
- A Sourcegraph access token
- Access to a Sourcegraph instance (default: sourcegraph.com)

## Quick Start

1. **Clone or navigate to the project:**
   ```bash
   cd nlsearch-app
   ```

2. **Set up environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env and add your Sourcegraph token
   ```

   Or export directly:
   ```bash
   export SOURCEGRAPH_TOKEN="your_access_token_here"
   export SOURCEGRAPH_URL="https://sourcegraph.com"  # optional
   export PORT="8080"  # optional
   ```

3. **Run the backend:**
   ```bash
   cd backend
   go run main.go
   ```

4. **Open your browser:**
   Navigate to `http://localhost:8080`

## Configuration

Configure the app using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SOURCEGRAPH_TOKEN` | Your Sourcegraph access token | **Required** |
| `SOURCEGRAPH_URL` | Sourcegraph instance URL | `https://sourcegraph.com` |
| `PORT` | Server port | `8080` |

## Getting a Sourcegraph Token

1. Go to your Sourcegraph instance (e.g., https://sourcegraph.com)
2. Navigate to Settings → Access tokens
3. Click "Generate new token"
4. Give it a name (e.g., "NLSearch App")
5. Copy the token and use it in your configuration

## Usage

1. Type your natural language query in the search box
2. Click "Search" or press Enter
3. Wait for the Deep Search API to process your query
4. View the answer and sources

### Example Queries

- "all repos which have python files"
- "find all TypeScript interfaces in the frontend"
- "show me error handling patterns in Go code"
- "repositories using React hooks"

## Project Structure

```
nlsearch-app/
├── backend/
│   ├── main.go          # Go backend server with API endpoints
│   └── go.mod           # Go module definition
├── frontend/
│   └── index.html       # Web UI (HTML/CSS/JS)
├── .env.example         # Example environment variables
└── README.md            # This file
```

## API Endpoints

### POST `/api/query`

Submit a natural language query.

**Request:**
```json
{
  "query": "all repos which have python files"
}
```

**Response:**
```json
{
  "answer": "Here are the repositories...",
  "sources": [
    {
      "type": "Repository",
      "label": "github.com/example/repo"
    }
  ]
}
```

### GET `/health`

Health check endpoint.

## How It Works

1. User submits a natural language query via the web UI
2. Frontend sends the query to the backend API
3. Backend creates a Deep Search conversation with the Sourcegraph API
4. Backend polls for completion (up to 60 seconds)
5. Result is returned to the frontend and displayed

## Development

### Running in Development Mode

The backend serves both the API and the frontend static files. Any changes to the frontend HTML will be reflected immediately on refresh.

For backend changes, restart the Go server:
```bash
cd backend
go run main.go
```

### Building for Production

```bash
cd backend
go build -o nlsearch-server main.go
```

Then run:
```bash
SOURCEGRAPH_TOKEN=your_token ./nlsearch-server
```

## Troubleshooting

**"SOURCEGRAPH_TOKEN environment variable is required"**
- Make sure you've set the `SOURCEGRAPH_TOKEN` environment variable

**"Failed to create conversation: unexpected status 401"**
- Your access token is invalid or expired
- Generate a new token from your Sourcegraph instance

**"timeout waiting for response"**
- The Deep Search query is taking too long
- Try a simpler query
- The timeout is currently set to 60 seconds

**"Network error"**
- Check your internet connection
- Verify the `SOURCEGRAPH_URL` is correct
- Ensure the Sourcegraph instance is accessible

## License

MIT License - feel free to use and modify as needed.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.
