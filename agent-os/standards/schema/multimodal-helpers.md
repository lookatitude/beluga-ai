# Multimodal Helpers

**Pattern:** For each multimodal type provide:

- `AsXxx(msg) (*Xxx, bool)` — type assert (or metadata-based check when the value is a struct).
- `IsXxx(msg) bool` — convenience that uses `AsXxx` and returns the bool.

**Document / struct:** When the value is a struct (e.g. `Document`) and a real type assert is not possible, infer from metadata (e.g. `audio_url`, `audio_data`, `transcript`) and document this as a known limitation.

**Extras:** `HasMultimodalContent(msg)`, `HasMultimodalDocument(doc)`, `ExtractMultimodalData(msg)` when useful.
