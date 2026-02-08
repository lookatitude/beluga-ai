package eval

import (
	"context"
	"encoding/json"
	"os"
)

// Dataset is a named collection of evaluation samples.
type Dataset struct {
	// Name is a human-readable identifier for this dataset.
	Name string `json:"name"`
	// Samples is the collection of evaluation samples.
	Samples []EvalSample `json:"samples"`
}

// LoadDataset reads a dataset from a JSON file at the given path.
func LoadDataset(path string) (*Dataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ds Dataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return nil, err
	}
	return &ds, nil
}

// Save writes the dataset to a JSON file at the given path.
func (d *Dataset) Save(path string) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Augmenter generates additional evaluation samples from an existing sample.
// Implementations may use an LLM to paraphrase, add noise, or create adversarial
// variants for more robust evaluation.
type Augmenter interface {
	// Augment generates one or more new samples derived from the given sample.
	Augment(ctx context.Context, sample EvalSample) ([]EvalSample, error)
}
