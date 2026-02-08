package tool

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestNewStaticMCPRegistry_Empty(t *testing.T) {
	reg := NewStaticMCPRegistry()
	servers, err := reg.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
}

func TestNewStaticMCPRegistry_WithServers(t *testing.T) {
	s1 := MCPServerInfo{Name: "server1", URL: "http://s1.example.com", Transport: "streamable-http"}
	s2 := MCPServerInfo{Name: "server2", URL: "http://s2.example.com", Transport: "streamable-http"}

	reg := NewStaticMCPRegistry(s1, s2)
	servers, err := reg.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	if servers[0].Name != "server1" {
		t.Errorf("servers[0].Name = %q, want %q", servers[0].Name, "server1")
	}
	if servers[1].Name != "server2" {
		t.Errorf("servers[1].Name = %q, want %q", servers[1].Name, "server2")
	}
}

func TestStaticMCPRegistry_Discover_ReturnsCopy(t *testing.T) {
	s1 := MCPServerInfo{Name: "original", URL: "http://orig.example.com"}
	reg := NewStaticMCPRegistry(s1)

	servers, _ := reg.Discover(context.Background())
	servers[0].Name = "modified"

	// Original should be unchanged.
	servers2, _ := reg.Discover(context.Background())
	if servers2[0].Name != "original" {
		t.Errorf("Discover should return a copy; got %q, want %q", servers2[0].Name, "original")
	}
}

func TestStaticMCPRegistry_Search_MatchesByName(t *testing.T) {
	tests := []struct {
		name    string
		servers []MCPServerInfo
		query   string
		want    int
	}{
		{
			name: "exact match",
			servers: []MCPServerInfo{
				{Name: "weather", URL: "http://weather.example.com"},
				{Name: "search", URL: "http://search.example.com"},
			},
			query: "weather",
			want:  1,
		},
		{
			name: "partial match",
			servers: []MCPServerInfo{
				{Name: "weather-api", URL: "http://weather.example.com"},
				{Name: "search-engine", URL: "http://search.example.com"},
			},
			query: "weather",
			want:  1,
		},
		{
			name: "case insensitive match",
			servers: []MCPServerInfo{
				{Name: "Weather-API", URL: "http://weather.example.com"},
				{Name: "SEARCH", URL: "http://search.example.com"},
			},
			query: "weather",
			want:  1,
		},
		{
			name: "multiple matches",
			servers: []MCPServerInfo{
				{Name: "weather-us", URL: "http://us.example.com"},
				{Name: "weather-eu", URL: "http://eu.example.com"},
				{Name: "search", URL: "http://search.example.com"},
			},
			query: "weather",
			want:  2,
		},
		{
			name: "no matches",
			servers: []MCPServerInfo{
				{Name: "weather", URL: "http://weather.example.com"},
			},
			query: "calendar",
			want:  0,
		},
		{
			name: "empty query matches all",
			servers: []MCPServerInfo{
				{Name: "a", URL: "http://a.example.com"},
				{Name: "b", URL: "http://b.example.com"},
			},
			query: "",
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewStaticMCPRegistry(tt.servers...)
			results, err := reg.Search(context.Background(), tt.query)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if len(results) != tt.want {
				t.Errorf("Search(%q) returned %d results, want %d", tt.query, len(results), tt.want)
			}
		})
	}
}

func TestStaticMCPRegistry_Search_CaseInsensitiveQuery(t *testing.T) {
	reg := NewStaticMCPRegistry(
		MCPServerInfo{Name: "MyWeather", URL: "http://example.com"},
	)

	for _, q := range []string{"myweather", "MYWEATHER", "MyWeather", "myWeather"} {
		results, err := reg.Search(context.Background(), q)
		if err != nil {
			t.Fatalf("Search(%q) error = %v", q, err)
		}
		if len(results) != 1 {
			t.Errorf("Search(%q) returned %d results, want 1", q, len(results))
		}
	}
}

func TestMCPServerInfo_Fields(t *testing.T) {
	tools := []schema.ToolDefinition{
		{Name: "tool1", Description: "Tool 1"},
		{Name: "tool2", Description: "Tool 2"},
	}

	info := MCPServerInfo{
		Name:      "test-server",
		URL:       "http://test.example.com",
		Tools:     tools,
		Transport: "streamable-http",
	}

	if info.Name != "test-server" {
		t.Errorf("Name = %q, want %q", info.Name, "test-server")
	}
	if info.URL != "http://test.example.com" {
		t.Errorf("URL = %q, want %q", info.URL, "http://test.example.com")
	}
	if len(info.Tools) != 2 {
		t.Errorf("Tools count = %d, want 2", len(info.Tools))
	}
	if info.Transport != "streamable-http" {
		t.Errorf("Transport = %q, want %q", info.Transport, "streamable-http")
	}
}

func TestContainsCI(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "Hello World", true},
		{"Hello", "Hello World", false},
		{"Hello", "", true},
		{"", "", true},
		{"", "a", false},
		{"ABC", "abc", true},
		{"abc", "ABC", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := containsCI(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsCI(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestStaticMCPRegistry_ImplementsMCPRegistry(t *testing.T) {
	var _ MCPRegistry = (*StaticMCPRegistry)(nil)
}
