# Risk Scoring

Additive risk score determines content safety.

## Scoring Weights
```go
toxicity patterns matched: +0.4
bias patterns matched:     +0.2
harmful patterns matched:  +0.5
```

## Safety Threshold
```go
result.Safe = result.RiskScore < 0.3
```
- Content with score >= 0.3 is unsafe
- Multiple issues of same type don't stack (e.g., 2 toxicity matches = still +0.4)
- But different types stack (toxicity + harmful = 0.9)

## Pattern Types
```go
toxicityPatterns  // Slurs, violent words, profanity
biasPatterns      // Overgeneralizations, false certainty
harmfulPatterns   // Instructions for illegal/dangerous activities
```

## Future: Configurability
Thresholds and weights should be configurable per use case (not yet implemented).
