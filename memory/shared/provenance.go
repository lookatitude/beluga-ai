package shared

import (
	"crypto/sha256"
	"time"
)

// Provenance records the cryptographic lineage of a fragment write.
// Each provenance entry contains a SHA-256 hash of the content, the
// author who performed the write, and a reference to the parent hash
// forming an immutable chain.
type Provenance struct {
	// ContentHash is the SHA-256 digest of the fragment content at this version.
	ContentHash [32]byte

	// AuthorID identifies the agent that performed the write.
	AuthorID string

	// WrittenAt is the time the write occurred.
	WrittenAt time.Time

	// ParentHash is the ContentHash of the previous version. A zero value
	// indicates this is the first version.
	ParentHash [32]byte
}

// ComputeProvenance creates a new Provenance for the given content and author,
// chaining from the provided parent hash. The content hash is computed using
// SHA-256.
func ComputeProvenance(content string, authorID string, parentHash [32]byte) *Provenance {
	return &Provenance{
		ContentHash: sha256.Sum256([]byte(content)),
		AuthorID:    authorID,
		WrittenAt:   time.Now(),
		ParentHash:  parentHash,
	}
}

// Verify checks that the ContentHash matches the SHA-256 digest of the given
// content. It returns true if the content is authentic, false otherwise.
func (p *Provenance) Verify(content string) bool {
	expected := sha256.Sum256([]byte(content))
	return p.ContentHash == expected
}
