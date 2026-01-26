# Document and VoiceDocument

**Document:** `PageContent`, `Metadata`, `ID`, `Embedding`, `Score`. `NewDocument(pageContent, metadata)`, `NewDocumentWithID(id, pageContent, metadata)`.

**Message:** `Document` must implement `Message`: `GetType` → `RoleSystem`, `GetContent` → `PageContent`, `ToolCalls` → `nil`, `AdditionalArgs` → `nil`. Enables use in pipelines that accept `Message`.

**VoiceDocument:** Embeds `Document`; adds `AudioURL`, `Transcript`, `AudioData`, etc. Override `GetContent`: return `Transcript` when present, otherwise `PageContent`. For document variants: embed the base and override only what changes.
