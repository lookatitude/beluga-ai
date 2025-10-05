package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProfileManager implements performance profile management and trend analysis
type ProfileManager struct {
	options  ProfileManagerOptions
	mu       sync.RWMutex
	profiles map[string]*PerformanceProfile // key: provider:model
}

// NewProfileManager creates a new profile manager with the specified options
func NewProfileManager(options ProfileManagerOptions) (*ProfileManager, error) {
	if options.StorageType == "" {
		options.StorageType = "memory"
	}
	if options.MaxProfiles == 0 {
		options.MaxProfiles = 1000
	}
	if options.ArchiveAfter == 0 {
		options.ArchiveAfter = 30 * 24 * time.Hour // 30 days
	}

	return &ProfileManager{
		options:  options,
		profiles: make(map[string]*PerformanceProfile),
	}, nil
}

// CreateProfile creates a new performance profile for a provider/model combination
func (pm *ProfileManager) CreateProfile(ctx context.Context, providerName, modelName string) (*PerformanceProfile, error) {
	if providerName == "" {
		return nil, fmt.Errorf("provider name cannot be empty")
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	profileKey := fmt.Sprintf("%s:%s", providerName, modelName)

	// Check if profile already exists
	if _, exists := pm.profiles[profileKey]; exists {
		return nil, fmt.Errorf("profile already exists for provider %s, model %s", providerName, modelName)
	}

	// Check profile limit
	if len(pm.profiles) >= pm.options.MaxProfiles {
		return nil, fmt.Errorf("maximum number of profiles (%d) reached", pm.options.MaxProfiles)
	}

	// Create new profile
	profile := &PerformanceProfile{
		ProfileID:        fmt.Sprintf("profile-%s-%s-%d", providerName, modelName, time.Now().UnixNano()),
		ProviderName:     providerName,
		ModelName:        modelName,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		BenchmarkResults: make([]*BenchmarkResult, 0),
	}

	pm.profiles[profileKey] = profile
	return profile, nil
}

// UpdateProfile adds new benchmark results to an existing performance profile
func (pm *ProfileManager) UpdateProfile(ctx context.Context, profileID string, result *BenchmarkResult) error {
	if profileID == "" {
		return fmt.Errorf("profile ID cannot be empty")
	}
	if result == nil {
		return fmt.Errorf("benchmark result cannot be nil")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Find profile by ID
	var targetProfile *PerformanceProfile
	for _, profile := range pm.profiles {
		if profile.ProfileID == profileID {
			targetProfile = profile
			break
		}
	}

	if targetProfile == nil {
		return fmt.Errorf("profile with ID %s not found", profileID)
	}

	// Validate result matches profile
	if result.ProviderName != targetProfile.ProviderName {
		return fmt.Errorf("result provider %s does not match profile provider %s",
			result.ProviderName, targetProfile.ProviderName)
	}
	if result.ModelName != targetProfile.ModelName {
		return fmt.Errorf("result model %s does not match profile model %s",
			result.ModelName, targetProfile.ModelName)
	}

	// Add result to profile
	targetProfile.BenchmarkResults = append(targetProfile.BenchmarkResults, result)
	targetProfile.UpdatedAt = time.Now()

	// Update trend analysis if enabled
	if pm.options.EnableTrends && len(targetProfile.BenchmarkResults) >= 3 {
		trendAnalysis, err := pm.calculateTrendAnalysis(targetProfile.BenchmarkResults)
		if err == nil {
			targetProfile.TrendAnalysis = trendAnalysis
		}
	}

	return nil
}

// GetProfile retrieves a performance profile by provider and model
func (pm *ProfileManager) GetProfile(ctx context.Context, providerName, modelName string) (*PerformanceProfile, error) {
	if providerName == "" {
		return nil, fmt.Errorf("provider name cannot be empty")
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	profileKey := fmt.Sprintf("%s:%s", providerName, modelName)
	profile, exists := pm.profiles[profileKey]
	if !exists {
		return nil, fmt.Errorf("profile not found for provider %s, model %s", providerName, modelName)
	}

	// Return a copy to prevent external modification
	profileCopy := *profile
	profileCopy.BenchmarkResults = make([]*BenchmarkResult, len(profile.BenchmarkResults))
	copy(profileCopy.BenchmarkResults, profile.BenchmarkResults)

	if profile.TrendAnalysis != nil {
		trendCopy := *profile.TrendAnalysis
		profileCopy.TrendAnalysis = &trendCopy
	}

	return &profileCopy, nil
}

// ListProfiles returns all available performance profiles with optional filtering
func (pm *ProfileManager) ListProfiles(ctx context.Context, filter ProfileFilter) ([]*PerformanceProfile, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var filteredProfiles []*PerformanceProfile

	for _, profile := range pm.profiles {
		if pm.matchesFilter(profile, filter) {
			// Create copy to prevent external modification
			profileCopy := *profile
			filteredProfiles = append(filteredProfiles, &profileCopy)
		}
	}

	return filteredProfiles, nil
}

// ArchiveOldResults removes or archives old benchmark results to manage storage
func (pm *ProfileManager) ArchiveOldResults(ctx context.Context, olderThan time.Time) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	archivedCount := 0

	for _, profile := range pm.profiles {
		// Filter out old results
		var keepResults []*BenchmarkResult
		for _, result := range profile.BenchmarkResults {
			if result.Timestamp.After(olderThan) {
				keepResults = append(keepResults, result)
			} else {
				archivedCount++
			}
		}

		// Update profile with remaining results
		profile.BenchmarkResults = keepResults

		// Recalculate trend analysis if results were removed
		if pm.options.EnableTrends && len(keepResults) >= 3 {
			trendAnalysis, err := pm.calculateTrendAnalysis(keepResults)
			if err == nil {
				profile.TrendAnalysis = trendAnalysis
			}
		} else {
			profile.TrendAnalysis = nil
		}
	}

	return nil
}

// Private helper methods

func (pm *ProfileManager) matchesFilter(profile *PerformanceProfile, filter ProfileFilter) bool {
	// Check provider name filter
	if len(filter.ProviderNames) > 0 {
		found := false
		for _, name := range filter.ProviderNames {
			if profile.ProviderName == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check model name filter
	if len(filter.ModelNames) > 0 {
		found := false
		for _, name := range filter.ModelNames {
			if profile.ModelName == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check creation time filter
	if !filter.CreatedAfter.IsZero() && profile.CreatedAt.Before(filter.CreatedAfter) {
		return false
	}

	// Check update time filter
	if !filter.UpdatedAfter.IsZero() && profile.UpdatedAt.Before(filter.UpdatedAfter) {
		return false
	}

	return true
}

func (pm *ProfileManager) calculateTrendAnalysis(results []*BenchmarkResult) (*TrendAnalysis, error) {
	if len(results) < 3 {
		return nil, fmt.Errorf("insufficient data for trend analysis")
	}

	// Sort results by timestamp
	sortedResults := make([]*BenchmarkResult, len(results))
	copy(sortedResults, results)

	for i := 0; i < len(sortedResults)-1; i++ {
		for j := 0; j < len(sortedResults)-i-1; j++ {
			if sortedResults[j].Timestamp.After(sortedResults[j+1].Timestamp) {
				sortedResults[j], sortedResults[j+1] = sortedResults[j+1], sortedResults[j]
			}
		}
	}

	trends := &TrendAnalysis{
		TrendID:         fmt.Sprintf("trend-%d", time.Now().UnixNano()),
		DataPoints:      len(results),
		ConfidenceLevel: 0.85, // Default confidence level
	}

	// Analyze latency trend
	trends.LatencyTrend = pm.analyzeTrendDirection(sortedResults, func(r *BenchmarkResult) float64 {
		return float64(r.LatencyMetrics.Mean.Milliseconds())
	})

	// Analyze throughput trend
	trends.ThroughputTrend = pm.analyzeTrendDirection(sortedResults, func(r *BenchmarkResult) float64 {
		return r.ThroughputRPS
	})

	// Analyze cost trend
	trends.CostTrend = pm.analyzeTrendDirection(sortedResults, func(r *BenchmarkResult) float64 {
		return r.CostAnalysis.TotalCostUSD
	})

	// Generate summary
	trends.TrendSummary = pm.generateTrendSummary(trends)

	return trends, nil
}

func (pm *ProfileManager) analyzeTrendDirection(results []*BenchmarkResult, valueExtractor func(*BenchmarkResult) float64) string {
	if len(results) < 3 {
		return "insufficient_data"
	}

	values := make([]float64, len(results))
	for i, result := range results {
		values[i] = valueExtractor(result)
	}

	// Simple linear trend analysis
	firstThird := pm.calculateMean(values[:len(values)/3])
	lastThird := pm.calculateMean(values[len(values)*2/3:])

	threshold := 0.05 // 5% change threshold

	if lastThird > firstThird*(1+threshold) {
		return "increasing"
	} else if lastThird < firstThird*(1-threshold) {
		return "decreasing"
	} else {
		return "stable"
	}
}

func (pm *ProfileManager) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (pm *ProfileManager) generateTrendSummary(trends *TrendAnalysis) string {
	improving := 0
	degrading := 0
	stable := 0

	trendMap := map[string]string{
		"latency":    trends.LatencyTrend,
		"throughput": trends.ThroughputTrend,
		"cost":       trends.CostTrend,
	}

	for metric, trend := range trendMap {
		switch trend {
		case "decreasing": // For latency and cost, decreasing is improving
			if metric == "latency" || metric == "cost" {
				improving++
			} else {
				degrading++
			}
		case "increasing": // For throughput, increasing is improving
			if metric == "throughput" {
				improving++
			} else {
				degrading++
			}
		case "stable":
			stable++
		}
	}

	if improving > degrading {
		return fmt.Sprintf("Overall improvement: %d metrics improving, %d stable, %d degrading",
			improving, stable, degrading)
	} else if degrading > improving {
		return fmt.Sprintf("Performance degradation: %d metrics degrading, %d stable, %d improving",
			degrading, stable, improving)
	} else {
		return fmt.Sprintf("Stable performance: %d metrics stable, %d improving, %d degrading",
			stable, improving, degrading)
	}
}
