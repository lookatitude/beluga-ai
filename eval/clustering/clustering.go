package clustering

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

// SimilarityScorer computes the similarity between two conversations.
// The result must be in [0.0, 1.0] where 1.0 means identical.
type SimilarityScorer interface {
	// Score returns a similarity score in [0.0, 1.0] between two conversations.
	Score(ctx context.Context, a, b Conversation) (float64, error)
}

// PatternDetector identifies recurring patterns across conversations.
type PatternDetector interface {
	// Detect finds patterns in a set of conversations and returns them.
	Detect(ctx context.Context, convs []Conversation) ([]Pattern, error)
}

// ConversationClusterer groups conversations into clusters based on similarity.
type ConversationClusterer interface {
	// Cluster groups the given conversations into clusters.
	Cluster(ctx context.Context, convs []Conversation) ([]Cluster, error)
}

// Conversation represents a single agent conversation for clustering analysis.
type Conversation struct {
	// ID uniquely identifies this conversation.
	ID string
	// Turns contains the ordered conversation turns.
	Turns []Turn
	// Metadata holds arbitrary key-value pairs for this conversation.
	Metadata map[string]any
}

// Turn represents a single turn in a conversation.
type Turn struct {
	// Role is the speaker role (e.g., "user", "assistant").
	Role string
	// Content is the text content of this turn.
	Content string
}

// Cluster represents a group of similar conversations.
type Cluster struct {
	// ID uniquely identifies this cluster.
	ID string
	// Label is a human-readable description of the cluster.
	Label string
	// Conversations are the members of this cluster.
	Conversations []Conversation
	// Centroid is the representative conversation closest to the center.
	Centroid *Conversation
}

// Pattern represents a recurring interaction pattern detected across conversations.
type Pattern struct {
	// Name is a short label for this pattern.
	Name string
	// Description explains the pattern.
	Description string
	// Frequency is the number of conversations exhibiting this pattern.
	Frequency int
	// Examples are conversation IDs that exhibit this pattern.
	Examples []string
}

// Factory creates a ConversationClusterer from a Config.
type Factory func(cfg Config) (ConversationClusterer, error)

// Config holds configuration for creating a ConversationClusterer.
type Config struct {
	// MinClusterSize is the minimum number of conversations in a cluster.
	MinClusterSize int
	// MaxClusters is the maximum number of clusters to produce.
	MaxClusters int
	// Threshold is the similarity threshold for merging clusters.
	Threshold float64
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a clusterer factory to the global registry.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a ConversationClusterer by name from the registry.
func New(name string, cfg Config) (ConversationClusterer, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("clustering: unknown clusterer %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered clusterers, sorted alphabetically.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Option configures a default clusterer.
type Option func(*options)

type options struct {
	metric      SimilarityScorer
	threshold   float64
	maxClusters int
	minSize     int
}

// WithMetric sets the similarity scorer for clustering.
func WithMetric(m SimilarityScorer) Option {
	return func(o *options) { o.metric = m }
}

// WithThreshold sets the similarity threshold for merging clusters.
func WithThreshold(t float64) Option {
	return func(o *options) { o.threshold = t }
}

// WithMaxClusters sets the maximum number of clusters.
func WithMaxClusters(n int) Option {
	return func(o *options) { o.maxClusters = n }
}

// WithMinClusterSize sets the minimum cluster size.
func WithMinClusterSize(n int) Option {
	return func(o *options) { o.minSize = n }
}

// NewAgglomerative creates an agglomerative (bottom-up) hierarchical clusterer.
func NewAgglomerative(opts ...Option) *AgglomerativeClusterer {
	o := &options{
		metric:      &JaccardSimilarity{},
		threshold:   0.5,
		maxClusters: 10,
		minSize:     1,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &AgglomerativeClusterer{opts: *o}
}

// AgglomerativeClusterer implements ConversationClusterer using agglomerative
// hierarchical clustering with configurable similarity metrics.
type AgglomerativeClusterer struct {
	opts options
}

var _ ConversationClusterer = (*AgglomerativeClusterer)(nil)

// Cluster groups conversations using agglomerative hierarchical clustering.
func (c *AgglomerativeClusterer) Cluster(ctx context.Context, convs []Conversation) ([]Cluster, error) {
	if len(convs) == 0 {
		return nil, nil
	}

	// Initialize: each conversation is its own cluster.
	clusters := make([]Cluster, len(convs))
	for i, conv := range convs {
		clusters[i] = Cluster{
			ID:            fmt.Sprintf("cluster-%d", i),
			Conversations: []Conversation{conv},
			Centroid:      &convs[i],
		}
	}

	for len(clusters) > 1 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		bestI, bestJ, bestSim, err := c.findClosestPair(ctx, clusters)
		if err != nil {
			return nil, err
		}

		// Stop when similarity drops below threshold, unless we still exceed the cap.
		belowThreshold := bestSim < c.opts.threshold
		atOrUnderCap := c.opts.maxClusters <= 0 || len(clusters) <= c.opts.maxClusters
		if belowThreshold && atOrUnderCap {
			break
		}

		// Merge bestJ into bestI.
		merged := mergeClusters(clusters[bestI], clusters[bestJ])
		clusters[bestI] = merged
		clusters = append(clusters[:bestJ], clusters[bestJ+1:]...)
	}

	// Filter by minimum size.
	var result []Cluster
	for _, cl := range clusters {
		if len(cl.Conversations) >= c.opts.minSize {
			result = append(result, cl)
		}
	}

	return result, nil
}

func (c *AgglomerativeClusterer) findClosestPair(ctx context.Context, clusters []Cluster) (int, int, float64, error) {
	bestI, bestJ := 0, 1
	bestSim := -1.0

	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			if err := ctx.Err(); err != nil {
				return 0, 0, 0, err
			}
			sim, err := c.clusterSimilarity(ctx, clusters[i], clusters[j])
			if err != nil {
				return 0, 0, 0, err
			}
			if sim > bestSim {
				bestSim = sim
				bestI = i
				bestJ = j
			}
		}
	}

	return bestI, bestJ, bestSim, nil
}

func (c *AgglomerativeClusterer) clusterSimilarity(ctx context.Context, a, b Cluster) (float64, error) {
	// Average linkage: mean similarity between all pairs.
	var total float64
	var count int
	for _, ca := range a.Conversations {
		for _, cb := range b.Conversations {
			sim, err := c.opts.metric.Score(ctx, ca, cb)
			if err != nil {
				return 0, err
			}
			total += sim
			count++
		}
	}
	if count == 0 {
		return 0, nil
	}
	return total / float64(count), nil
}

func mergeClusters(a, b Cluster) Cluster {
	convs := make([]Conversation, 0, len(a.Conversations)+len(b.Conversations))
	convs = append(convs, a.Conversations...)
	convs = append(convs, b.Conversations...)
	return Cluster{
		ID:            a.ID,
		Label:         a.Label,
		Conversations: convs,
		Centroid:      a.Centroid,
	}
}

// JaccardSimilarity computes similarity based on shared vocabulary between conversations.
type JaccardSimilarity struct{}

var _ SimilarityScorer = (*JaccardSimilarity)(nil)

// Score returns the Jaccard similarity of the word sets of two conversations.
func (j *JaccardSimilarity) Score(_ context.Context, a, b Conversation) (float64, error) {
	wordsA := extractWords(a)
	wordsB := extractWords(b)

	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0, nil
	}

	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}

	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0, nil
	}

	return float64(intersection) / float64(union), nil
}

func extractWords(c Conversation) map[string]bool {
	words := make(map[string]bool)
	for _, t := range c.Turns {
		for _, w := range splitWords(t.Content) {
			if len(w) > 0 {
				words[w] = true
			}
		}
	}
	return words
}

func splitWords(s string) []string {
	var words []string
	var current []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			current = append(current, c|0x20) // lowercase
		} else if len(current) > 0 {
			words = append(words, string(current))
			current = current[:0]
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// TurnPatternDetector detects patterns based on turn structure (e.g., greeting patterns,
// multi-turn Q&A patterns).
type TurnPatternDetector struct{}

var _ PatternDetector = (*TurnPatternDetector)(nil)

// Detect identifies recurring turn patterns across conversations.
func (d *TurnPatternDetector) Detect(_ context.Context, convs []Conversation) ([]Pattern, error) {
	if len(convs) == 0 {
		return nil, nil
	}

	patterns := make(map[string]*Pattern)
	addTurnCountPatterns(convs, patterns)
	addRoleSequencePatterns(convs, patterns)
	addLengthOutlierPatterns(convs, patterns)

	result := make([]Pattern, 0, len(patterns))
	for _, p := range patterns {
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// addTurnCountPatterns groups conversations by turn count and records
// patterns where at least two conversations share the same count.
func addTurnCountPatterns(convs []Conversation, patterns map[string]*Pattern) {
	turnCounts := make(map[int][]string)
	for _, c := range convs {
		turnCounts[len(c.Turns)] = append(turnCounts[len(c.Turns)], c.ID)
	}
	for count, ids := range turnCounts {
		if len(ids) < 2 {
			continue
		}
		name := fmt.Sprintf("%d-turn-conversation", count)
		patterns[name] = &Pattern{
			Name:        name,
			Description: fmt.Sprintf("Conversations with exactly %d turns", count),
			Frequency:   len(ids),
			Examples:    ids,
		}
	}
}

// addRoleSequencePatterns groups conversations by role sequence and
// records patterns where the same sequence appears twice or more.
func addRoleSequencePatterns(convs []Conversation, patterns map[string]*Pattern) {
	roleSeqs := make(map[string][]string)
	for _, c := range convs {
		seq := roleSequence(c)
		roleSeqs[seq] = append(roleSeqs[seq], c.ID)
	}
	for seq, ids := range roleSeqs {
		if len(ids) < 2 {
			continue
		}
		name := "role-pattern-" + seq
		patterns[name] = &Pattern{
			Name:        name,
			Description: fmt.Sprintf("Conversations with role sequence: %s", seq),
			Frequency:   len(ids),
			Examples:    ids,
		}
	}
}

// addLengthOutlierPatterns records "short" and "long" conversation
// patterns based on deviation from the mean turn count.
func addLengthOutlierPatterns(convs []Conversation, patterns map[string]*Pattern) {
	shortIDs, longIDs := classifyByLength(convs)
	if len(shortIDs) >= 2 {
		patterns["short-conversation"] = &Pattern{
			Name:        "short-conversation",
			Description: "Conversations significantly shorter than average",
			Frequency:   len(shortIDs),
			Examples:    shortIDs,
		}
	}
	if len(longIDs) >= 2 {
		patterns["long-conversation"] = &Pattern{
			Name:        "long-conversation",
			Description: "Conversations significantly longer than average",
			Frequency:   len(longIDs),
			Examples:    longIDs,
		}
	}
}

// classifyByLength partitions conversations into short and long outliers
// relative to the mean turn count, using a sqrt(mean) threshold.
func classifyByLength(convs []Conversation) (shortIDs, longIDs []string) {
	var totalTurns int
	for _, c := range convs {
		totalTurns += len(c.Turns)
	}
	avgTurns := float64(totalTurns) / float64(len(convs))
	deviation := math.Sqrt(avgTurns)
	for _, c := range convs {
		turns := float64(len(c.Turns))
		switch {
		case turns < avgTurns-deviation:
			shortIDs = append(shortIDs, c.ID)
		case turns > avgTurns+deviation:
			longIDs = append(longIDs, c.ID)
		}
	}
	return shortIDs, longIDs
}

func roleSequence(c Conversation) string {
	parts := make([]string, len(c.Turns))
	for i, t := range c.Turns {
		parts[i] = t.Role
	}
	return strings.Join(parts, "-")
}

func init() {
	Register("agglomerative", func(cfg Config) (ConversationClusterer, error) {
		var opts []Option
		if cfg.Threshold > 0 {
			opts = append(opts, WithThreshold(cfg.Threshold))
		}
		if cfg.MaxClusters > 0 {
			opts = append(opts, WithMaxClusters(cfg.MaxClusters))
		}
		if cfg.MinClusterSize > 0 {
			opts = append(opts, WithMinClusterSize(cfg.MinClusterSize))
		}
		return NewAgglomerative(opts...), nil
	})
}
