package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const clientIdentifier = "nlsearch-app 1.0.0"

type Config struct {
	SourcegraphURL   string
	SourcegraphToken string
	Port             string
}

type DeepSearchClient struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

type CreateConversationRequest struct {
	Question string `json:"question"`
}

type Question struct {
	ID             int                      `json:"id"`
	ConversationID int                      `json:"conversation_id"`
	Question       string                   `json:"question"`
	Status         string                   `json:"status"`
	Answer         string                   `json:"answer,omitempty"`
	Sources        []map[string]interface{} `json:"sources,omitempty"`
	Stats          map[string]interface{}   `json:"stats"`
}

type Conversation struct {
	ID        int        `json:"id"`
	Questions []Question `json:"questions"`
}

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Answer  string                   `json:"answer"`
	Sources []map[string]interface{} `json:"sources,omitempty"`
	Error   string                   `json:"error,omitempty"`
}

func NewDeepSearchClient(baseURL, accessToken string) *DeepSearchClient {
	return &DeepSearchClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *DeepSearchClient) createConversation(ctx context.Context, question string) (*Conversation, error) {
	reqBody := CreateConversationRequest{Question: question}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/.api/deepsearch/v1", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.accessToken))
	req.Header.Set("X-Requested-With", clientIdentifier)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var conv Conversation
	if err := json.NewDecoder(resp.Body).Decode(&conv); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &conv, nil
}

func (c *DeepSearchClient) getConversation(ctx context.Context, conversationID int) (*Conversation, error) {
	apiURL := fmt.Sprintf("%s/.api/deepsearch/v1/%d", c.baseURL, conversationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.accessToken))
	req.Header.Set("X-Requested-With", clientIdentifier)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var conv Conversation
	if err := json.NewDecoder(resp.Body).Decode(&conv); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &conv, nil
}

func (c *DeepSearchClient) waitForCompletion(ctx context.Context, conversationID int, maxWait time.Duration) (*Question, error) {
	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for response")
			}

			conv, err := c.getConversation(ctx, conversationID)
			if err != nil {
				return nil, err
			}

			if len(conv.Questions) > 0 {
				q := conv.Questions[len(conv.Questions)-1]
				switch q.Status {
				case "completed":
					return &q, nil
				case "failed":
					return nil, fmt.Errorf("question processing failed")
				case "cancelled":
					return nil, fmt.Errorf("question was cancelled")
				}
			}
		}
	}
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	config := Config{
		SourcegraphURL:   getEnv("SOURCEGRAPH_URL", "https://sourcegraph.com"),
		SourcegraphToken: getEnv("SOURCEGRAPH_TOKEN", ""),
		Port:             getEnv("PORT", "8080"),
	}

	if config.SourcegraphToken == "" {
		log.Fatal("SOURCEGRAPH_TOKEN environment variable is required")
	}

	parsedURL, err := url.Parse(config.SourcegraphURL)
	if err != nil {
		log.Fatalf("Invalid SOURCEGRAPH_URL: %v", err)
	}
	config.SourcegraphURL = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	client := NewDeepSearchClient(config.SourcegraphURL, config.SourcegraphToken)

	http.HandleFunc("/api/query", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(QueryResponse{Error: "Invalid request body"})
			return
		}

		if req.Query == "" {
			json.NewEncoder(w).Encode(QueryResponse{Error: "Query is required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		prompt := fmt.Sprintf(`Convert this natural language request into a valid Sourcegraph search query. 

For guidance on proper syntax, refer to these files in github.com/sourcegraph/sourcegraph:
- internal/search/query/parser.go
- internal/search/query/validate.go
- internal/search/query/parser_test.go
- internal/search/query/validate_test.go
- client/branded/src/search-ui/components/QueryExamples.constants.ts

CRITICAL: Your response must be ONLY the search query itself. No explanations, no markdown, no code blocks, no additional text. Just the raw query string.

Request: %s`, req.Query)
		conv, err := client.createConversation(ctx, prompt)
		if err != nil {
			log.Printf("Error creating conversation: %v", err)
			json.NewEncoder(w).Encode(QueryResponse{Error: fmt.Sprintf("Failed to create conversation: %v", err)})
			return
		}

		question, err := client.waitForCompletion(ctx, conv.ID, 60*time.Second)
		if err != nil {
			log.Printf("Error waiting for completion: %v", err)
			json.NewEncoder(w).Encode(QueryResponse{Error: fmt.Sprintf("Failed to get response: %v", err)})
			return
		}

		query := extractQuery(question.Answer)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryResponse{
			Answer:  query,
			Sources: question.Sources,
		})
	}))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fs := http.FileServer(http.Dir("../frontend"))
	http.Handle("/", fs)

	log.Printf("Server starting on http://localhost:%s", config.Port)
	log.Printf("Using Sourcegraph instance: %s", config.SourcegraphURL)
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatal(err)
	}
}

func extractQuery(answer string) string {
	lines := strings.Split(strings.TrimSpace(answer), "\n")
	
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "For ") && !strings.HasPrefix(line, "Based ") {
			line = strings.Trim(line, "`")
			if (strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"")) ||
			   (strings.HasPrefix(line, "'") && strings.HasSuffix(line, "'")) {
				line = line[1 : len(line)-1]
			}
			return line
		}
	}
	
	if len(lines) > 0 {
		line := strings.TrimSpace(lines[len(lines)-1])
		line = strings.Trim(line, "`")
		if (strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"")) ||
		   (strings.HasPrefix(line, "'") && strings.HasSuffix(line, "'")) {
			line = line[1 : len(line)-1]
		}
		return line
	}
	
	return answer
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
