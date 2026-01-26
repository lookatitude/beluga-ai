# BaseMessage and Concrete Types

**BaseMessage:** `Content`. Concrete types embed it and implement `Message`.

**GetContent:** Always return a string. Use `Content` when set; for multimodal with no `Content`, use a fallback (e.g. `"[Image: url]"`, `"[Video: format data]"`) so text-only consumers have something.

**Multimodal and AdditionalArgs:** Put image/video/audio data in dedicated fields (e.g. `ImageURL`, `ImageData`) and also in `AdditionalArgs` (e.g. `image_url`, `image_data`) so both type-specific and generic code can access it.
