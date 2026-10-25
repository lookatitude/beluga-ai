package weaviate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore/internal/filterutil"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

const (
	// maxResponseBytes bounds every response body read from Weaviate.
	maxResponseBytes = 32 << 20
	// defaultTimeout is the timeout of the default HTTP client.
	defaultTimeout = 30 * time.Second
)

// classPattern is Weaviate's class-name grammar. The class is interpolated
// into GraphQL and URL paths, so anything else is rejected.
var classPattern = regexp.MustCompile(`^[A-Z][_0-9A-Za-z]*$`)

// idNamespace is the UUIDv5 namespace used to derive Weaviate object UUIDs
// from document IDs.
var idNamespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://github.com/lookatitude/beluga-ai/rag/vectorstore/weaviate"))

// reservedProps are object properties the store manages itself; document
// metadata must not overwrite them.
var reservedProps = map[string]bool{"content": true, "_beluga_id": true, "_additional": true}

func init() {
	vectorstore.Register("weaviate", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by the Weaviate vector database.
type Store struct {
	client  HTTPClient
	baseURL string
	apiKey  string
	class   string
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithClass sets the Weaviate class name.
func WithClass(name string) Option {
	return func(s *Store) { s.class = name }
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(s *Store) { s.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new Weaviate Store.
func New(baseURL string, opts ...Option) *Store {
	s := &Store{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: baseURL,
		class:   "Document",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("weaviate: base_url is required")
	}
	var opts []Option
	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if cls, ok := config.GetOption[string](cfg, "class"); ok {
		opts = append(opts, WithClass(cls))
	}
	return New(cfg.BaseURL, opts...), nil
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("weaviate: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}
	if err := s.validateClass("weaviate.add"); err != nil {
		return err
	}

	objects := make([]map[string]any, len(docs))
	for i, emb := range embeddings {
		doc := &docs[i]
		props := map[string]any{
			"content":    doc.Content,
			"_beluga_id": doc.ID,
		}
		for k, v := range doc.Metadata {
			if reservedProps[k] {
				return core.NewError("weaviate.add", core.ErrInvalidInput,
					fmt.Sprintf("document %q: metadata key %q is reserved", doc.ID, k), nil)
			}
			props[k] = v
		}

		objects[i] = map[string]any{
			"class":      s.class,
			"id":         uuidFromID(doc.ID),
			"properties": props,
			"vector":     float32SliceToFloat64(emb),
		}
	}

	body := map[string]any{
		"objects": objects,
	}
	respBody, err := s.doJSON(ctx, http.MethodPost, "/v1/batch/objects", body)
	if err != nil {
		return err
	}
	return batchErrors(respBody)
}

// batchErrors returns an error when a /v1/batch/objects response (HTTP 200)
// reports per-object failures.
func batchErrors(respBody []byte) error {
	var items []struct {
		ID     string `json:"id"`
		Result struct {
			Errors *struct {
				Error []struct {
					Message string `json:"message"`
				} `json:"error"`
			} `json:"errors"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &items); err != nil {
		return fmt.Errorf("weaviate: unmarshal batch response: %w", err)
	}
	var msgs []string
	for _, it := range items {
		if it.Result.Errors == nil {
			continue
		}
		for _, e := range it.Result.Errors.Error {
			msgs = append(msgs, fmt.Sprintf("%s: %s", it.ID, e.Message))
		}
	}
	if len(msgs) > 0 {
		return core.NewError("weaviate.add", core.ErrToolFailed,
			fmt.Sprintf("batch insert failed for %d object(s): %s", len(msgs), strings.Join(msgs, "; ")), nil)
	}
	return nil
}

// validateClass rejects class names that are not valid Weaviate identifiers.
func (s *Store) validateClass(op string) error {
	if !classPattern.MatchString(s.class) {
		return core.NewError(op, core.ErrInvalidInput,
			fmt.Sprintf("invalid class name %q: must match %s", s.class, classPattern.String()), nil)
	}
	return nil
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if err := s.validateClass("weaviate.search"); err != nil {
		return nil, err
	}
	whereClause, err := buildWhereClause(cfg.Filter)
	if err != nil {
		return nil, err
	}
	graphQL := buildGraphQLQuery(s.class, query, k, cfg.Threshold, whereClause)
	body := map[string]any{"query": graphQL}

	respBody, err := s.doJSON(ctx, http.MethodPost, "/v1/graphql", body)
	if err != nil {
		return nil, err
	}

	classResults, err := extractClassResults(respBody, s.class)
	if err != nil {
		return nil, err
	}
	if classResults == nil {
		return nil, nil
	}

	return parseClassResults(classResults, cfg.Threshold), nil
}

// buildGraphQLQuery constructs the GraphQL query for a Weaviate nearVector search.
func buildGraphQLQuery(class string, query []float32, k int, threshold float64, whereClause string) string {
	distanceClause := ""
	if threshold > 0 {
		distanceClause = fmt.Sprintf(`, distance: %f`, 1.0-threshold)
	}

	return fmt.Sprintf(`{
		Get {
			%s(
				limit: %d,
				nearVector: {vector: %s%s}
				%s
			) {
				content
				_beluga_id
				_additional {
					id
					distance
				}
			}
		}
	}`, class, k, float32SliceToJSONArray(query), distanceClause, whereClause)
}

// buildWhereClause constructs a Weaviate GraphQL where clause from filter
// metadata. Keys must be safe identifiers and values scalars; string values
// are emitted as JSON string literals (valid GraphQL string syntax), so a
// value can never terminate the literal and inject operands.
func buildWhereClause(filter map[string]any) (string, error) {
	if len(filter) == 0 {
		return "", nil
	}
	keys, err := filterutil.Validate("weaviate.search", filter)
	if err != nil {
		return "", err
	}
	operands := make([]string, 0, len(keys))
	for _, key := range keys {
		field, lit, err := whereValue(filter[key])
		if err != nil {
			return "", err
		}
		operands = append(operands, fmt.Sprintf(`{path:[%q],operator:Equal,%s:%s}`, key, field, lit))
	}
	if len(operands) == 1 {
		return fmt.Sprintf(`, where: %s`, operands[0]), nil
	}
	return fmt.Sprintf(`, where: {operator:And,operands:[%s]}`,
		joinStrings(operands, ",")), nil
}

// whereValue returns the Weaviate value field and literal for a scalar.
func whereValue(v any) (field, literal string, err error) {
	switch x := v.(type) {
	case string:
		b, err := json.Marshal(x)
		if err != nil {
			return "", "", fmt.Errorf("weaviate: encode filter value: %w", err)
		}
		return "valueText", string(b), nil
	case bool:
		return "valueBoolean", fmt.Sprintf("%t", x), nil
	case float32, float64:
		return "valueNumber", fmt.Sprintf("%v", x), nil
	default: // integer kinds; filterutil.Validate admits only scalars
		return "valueInt", fmt.Sprintf("%d", x), nil
	}
}

// extractClassResults parses the raw Weaviate GraphQL response and extracts the class results array.
func extractClassResults(respBody []byte, class string) ([]any, error) {
	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("weaviate: unmarshal response: %w", err)
	}
	if errs, ok := raw["errors"].([]any); ok && len(errs) > 0 {
		msgs := make([]string, 0, len(errs))
		for _, e := range errs {
			if m, ok := e.(map[string]any); ok {
				if msg, ok := m["message"].(string); ok {
					msgs = append(msgs, msg)
				}
			}
		}
		return nil, core.NewError("weaviate.search", core.ErrToolFailed,
			"graphql error: "+strings.Join(msgs, "; "), nil)
	}
	dataMap, ok := raw["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("weaviate: missing data in response")
	}
	getMap, ok := dataMap["Get"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("weaviate: missing Get in response")
	}
	classResults, ok := getMap[class].([]any)
	if !ok {
		return nil, nil
	}
	return classResults, nil
}

// parseClassResults converts raw Weaviate result objects into schema.Documents.
func parseClassResults(classResults []any, threshold float64) []schema.Document {
	docs := make([]schema.Document, 0, len(classResults))
	for _, item := range classResults {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		doc := weaviateObjectToDocument(obj)
		if threshold > 0 && doc.Score < threshold {
			continue
		}
		docs = append(docs, doc)
	}
	return docs
}

// weaviateObjectToDocument converts a single Weaviate result object to a schema.Document.
func weaviateObjectToDocument(obj map[string]any) schema.Document {
	doc := schema.Document{Metadata: make(map[string]any)}

	if id, ok := obj["_beluga_id"].(string); ok {
		doc.ID = id
	}
	if content, ok := obj["content"].(string); ok {
		doc.Content = content
	}
	if additional, ok := obj["_additional"].(map[string]any); ok {
		if dist, ok := additional["distance"].(float64); ok {
			doc.Score = 1.0 - dist
		}
		if doc.ID == "" {
			if id, ok := additional["id"].(string); ok {
				doc.ID = id
			}
		}
	}
	for k, v := range obj {
		if k != "content" && k != "_beluga_id" && k != "_additional" {
			doc.Metadata[k] = v
		}
	}
	return doc
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	if err := s.validateClass("weaviate.delete"); err != nil {
		return err
	}
	for _, id := range ids {
		_, err := s.doJSON(ctx, http.MethodDelete,
			fmt.Sprintf("/v1/objects/%s/%s", url.PathEscape(s.class), uuidFromID(id)), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("weaviate: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("weaviate: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weaviate: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("weaviate: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("weaviate: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// uuidFromID derives a deterministic, name-based UUIDv5 (SHA-1) from the
// full document ID, giving a stable, collision-resistant mapping from
// arbitrary IDs to Weaviate's UUID format.
//
// Objects written by releases before this change used a truncated hex
// encoding of the ID; re-index them (Add again, then delete the old objects
// by their legacy UUID) after upgrading.
func uuidFromID(id string) string {
	return uuid.NewSHA1(idNamespace, []byte(id)).String()
}

// float32SliceToFloat64 converts []float32 to []float64 for JSON serialization.
func float32SliceToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}

// float32SliceToJSONArray converts a float32 slice to a JSON array string.
func float32SliceToJSONArray(v []float32) string {
	data, _ := json.Marshal(float32SliceToFloat64(v))
	return string(data)
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
