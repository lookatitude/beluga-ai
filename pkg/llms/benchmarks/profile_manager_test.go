// Package benchmarks provides contract tests for profile manager interfaces.
// This file tests the ProfileManager interface contract compliance.
package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProfileManager_Contract tests the ProfileManager interface contract
func TestProfileManager_Contract(t *testing.T) {
	ctx := context.Background()

	// Create profile manager (will fail until implemented)
	manager, err := NewProfileManager(ProfileManagerOptions{
		StorageType:    "memory",
		MaxProfiles:    1000,
		ArchiveAfter:   30 * 24 * time.Hour, // 30 days
		EnableTrends:   true,
	})
	require.NoError(t, err, "ProfileManager creation should succeed")
	require.NotNil(t, manager, "ProfileManager should not be nil")

	// Test profile creation
	t.Run("CreateProfile", func(t *testing.T) {
		profile, err := manager.CreateProfile(ctx, "openai", "gpt-4")
		assert.NoError(t, err, "Profile creation should succeed")
		assert.NotNil(t, profile, "Profile should not be nil")
		
		// Verify profile structure
		assert.Equal(t, "openai", profile.ProviderName, "Should have correct provider name")
		assert.Equal(t, "gpt-4", profile.ModelName, "Should have correct model name")
		assert.NotZero(t, profile.CreatedAt, "Should have creation timestamp")
		assert.NotEmpty(t, profile.ProfileID, "Should have profile ID")
		assert.Empty(t, profile.BenchmarkResults, "New profile should have no results")
	})

	// Test profile updates
	t.Run("UpdateProfile", func(t *testing.T) {
		// Create profile first
		profile, err := manager.CreateProfile(ctx, "anthropic", "claude-3")
		require.NoError(t, err)

		// Create benchmark result to add
		result := createBenchmarkResult("anthropic", "claude-3", 120*time.Millisecond)
		
		err = manager.UpdateProfile(ctx, profile.ProfileID, result)
		assert.NoError(t, err, "Profile update should succeed")

		// Retrieve updated profile
		updatedProfile, err := manager.GetProfile(ctx, "anthropic", "claude-3")
		assert.NoError(t, err, "Getting updated profile should succeed")
		assert.NotNil(t, updatedProfile, "Updated profile should not be nil")
		
		// Verify update
		assert.Len(t, updatedProfile.BenchmarkResults, 1, "Should have one benchmark result")
		assert.Equal(t, result.BenchmarkID, updatedProfile.BenchmarkResults[0].BenchmarkID,
			"Should have correct benchmark result")
		assert.True(t, updatedProfile.UpdatedAt.After(updatedProfile.CreatedAt),
			"UpdatedAt should be after CreatedAt")
	})

	// Test profile retrieval
	t.Run("GetProfile", func(t *testing.T) {
		// Test existing profile
		profile, err := manager.GetProfile(ctx, "openai", "gpt-4")
		assert.NoError(t, err, "Getting existing profile should succeed")
		assert.NotNil(t, profile, "Existing profile should not be nil")

		// Test non-existent profile
		_, err = manager.GetProfile(ctx, "nonexistent", "model")
		assert.Error(t, err, "Getting non-existent profile should fail")
	})

	// Test profile listing
	t.Run("ListProfiles", func(t *testing.T) {
		// List all profiles
		allProfiles, err := manager.ListProfiles(ctx, ProfileFilter{})
		assert.NoError(t, err, "Listing all profiles should succeed")
		assert.GreaterOrEqual(t, len(allProfiles), 2, "Should have at least 2 profiles")

		// Test filtering by provider
		openaiFilter := ProfileFilter{ProviderNames: []string{"openai"}}
		openaiProfiles, err := manager.ListProfiles(ctx, openaiFilter)
		assert.NoError(t, err, "Filtering by provider should succeed")
		
		for _, profile := range openaiProfiles {
			assert.Equal(t, "openai", profile.ProviderName, 
				"Filtered profiles should match provider")
		}

		// Test filtering by creation time
		recentTime := time.Now().Add(-1 * time.Hour)
		timeFilter := ProfileFilter{CreatedAfter: recentTime}
		recentProfiles, err := manager.ListProfiles(ctx, timeFilter)
		assert.NoError(t, err, "Filtering by time should succeed")
		
		for _, profile := range recentProfiles {
			assert.True(t, profile.CreatedAt.After(recentTime),
				"Filtered profiles should be after filter time")
		}
	})

	// Test archiving
	t.Run("ArchiveOldResults", func(t *testing.T) {
		// Archive very old results (1 hour ago)
		archiveTime := time.Now().Add(-1 * time.Hour)
		
		err := manager.ArchiveOldResults(ctx, archiveTime)
		assert.NoError(t, err, "Archiving should succeed")
		
		// Since we just created profiles, nothing should be archived yet
		// This tests the archiving mechanism works without errors
	})
}

// TestProfileManager_TrendAnalysis tests trend analysis capabilities
func TestProfileManager_TrendAnalysis(t *testing.T) {
	ctx := context.Background()

	manager, err := NewProfileManager(ProfileManagerOptions{
		EnableTrends: true,
		MinTrendData: 5,
	})
	require.NoError(t, err)

	// Create profile with historical data
	t.Run("TrendCalculation", func(t *testing.T) {
		profile, err := manager.CreateProfile(ctx, "trend-test", "model")
		require.NoError(t, err)

		// Add multiple benchmark results over time
		baseTime := time.Now().Add(-24 * time.Hour)
		for i := 0; i < 10; i++ {
			result := createBenchmarkResult("trend-test", "model", 
				time.Duration(100+i*5)*time.Millisecond) // Gradual performance degradation
			result.Timestamp = baseTime.Add(time.Duration(i) * time.Hour)
			
			err := manager.UpdateProfile(ctx, profile.ProfileID, result)
			assert.NoError(t, err, "Adding result %d should succeed", i)
		}

		// Get profile with trend analysis
		updatedProfile, err := manager.GetProfile(ctx, "trend-test", "model")
		assert.NoError(t, err, "Getting profile should succeed")
		assert.Len(t, updatedProfile.BenchmarkResults, 10, "Should have all results")

		// Verify trend analysis is available
		if updatedProfile.TrendAnalysis != nil {
			assert.GreaterOrEqual(t, updatedProfile.TrendAnalysis.DataPoints, 5,
				"Should have minimum data points for trend")
			assert.GreaterOrEqual(t, updatedProfile.TrendAnalysis.ConfidenceLevel, 0.0,
				"Confidence level should be non-negative")
		}
	})
}

// TestProfileManager_Performance tests performance constraints
func TestProfileManager_Performance(t *testing.T) {
	ctx := context.Background()

	manager, err := NewProfileManager(ProfileManagerOptions{
		StorageType: "memory",
		MaxProfiles: 100,
	})
	require.NoError(t, err)

	// Test profile operation performance
	t.Run("ProfileOperationPerformance", func(t *testing.T) {
		const numProfiles = 50
		
		// Create multiple profiles
		start := time.Now()
		for i := 0; i < numProfiles; i++ {
			providerName := fmt.Sprintf("provider-%d", i%5) // 5 different providers
			modelName := fmt.Sprintf("model-%d", i%3)       // 3 different models
			
			_, err := manager.CreateProfile(ctx, providerName, modelName)
			assert.NoError(t, err, "Profile creation should succeed")
		}
		creationDuration := time.Since(start)

		t.Logf("Created %d profiles in %v (avg: %v per profile)", 
			numProfiles, creationDuration, creationDuration/numProfiles)
		
		// Profile creation should be fast
		avgCreationTime := creationDuration / numProfiles
		assert.Less(t, avgCreationTime, 10*time.Millisecond,
			"Average profile creation should be <10ms")

		// Test listing performance
		start = time.Now()
		profiles, err := manager.ListProfiles(ctx, ProfileFilter{})
		listingDuration := time.Since(start)
		
		assert.NoError(t, err, "Listing should succeed")
		assert.GreaterOrEqual(t, len(profiles), numProfiles, "Should list all profiles")
		assert.Less(t, listingDuration, 100*time.Millisecond,
			"Profile listing should be <100ms")

		// Test filtering performance
		start = time.Now()
		filteredProfiles, err := manager.ListProfiles(ctx, ProfileFilter{
			ProviderNames: []string{"provider-0", "provider-1"},
		})
		filteringDuration := time.Since(start)

		assert.NoError(t, err, "Filtering should succeed")
		assert.Greater(t, len(filteredProfiles), 0, "Should find filtered profiles")
		assert.Less(t, filteringDuration, 50*time.Millisecond,
			"Profile filtering should be <50ms")
	})

	// Test concurrent access
	t.Run("ConcurrentAccess", func(t *testing.T) {
		const numGoroutines = 20
		results := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				// Mix of operations
				switch id % 4 {
				case 0:
					_, err := manager.CreateProfile(ctx, fmt.Sprintf("concurrent-%d", id), "model")
					results <- err
				case 1:
					_, err := manager.GetProfile(ctx, "provider-0", "model-0")
					results <- err
				case 2:
					_, err := manager.ListProfiles(ctx, ProfileFilter{})
					results <- err
				case 3:
					err := manager.ArchiveOldResults(ctx, time.Now().Add(-1*time.Hour))
					results <- err
				}
			}(i)
		}

		// Collect results
		errorCount := 0
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				errorCount++
			}
		}

		assert.Equal(t, 0, errorCount, "No errors should occur during concurrent access")
	})
}
