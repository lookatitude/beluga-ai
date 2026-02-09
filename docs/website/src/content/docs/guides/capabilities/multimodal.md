---
title: Multimodal AI (Vision & Audio)
description: Learn how to process images, audio, and video with multimodal language models.
---

Modern AI models can understand more than text. Multimodal models process images, audio, and video, enabling applications like document scanning, video analysis, and audio intelligence. Beluga AI provides unified interfaces for multimodal AI across providers.

## What You'll Learn

This guide covers:
- Working with vision models for image analysis
- Processing audio for transcription and intelligence
- Combining text, image, and audio in single requests
- Provider-specific multimodal capabilities
- Building multimodal RAG systems
- Production patterns for multimodal AI

## When to Use Multimodal AI

Multimodal capabilities enable:
- **Document intelligence** (receipts, invoices, forms, IDs)
- **Visual Q&A** (charts, diagrams, screenshots)
- **Video analysis** (surveillance, content moderation, accessibility)
- **Audio intelligence** (meeting summaries, sentiment analysis)
- **Accessibility** (image descriptions, audio transcriptions)
- **Multimodal search** (find images by description, find audio by content)

## Prerequisites

Before starting this guide:
- Complete [Working with LLMs](/guides/working-with-llms)
- Understand `schema.Message` and content parts
- Have a multimodal-capable provider (OpenAI GPT-4o, Anthropic Claude 3, Google Gemini)

## Vision: Image Analysis

### Basic Image Analysis

Analyze images using vision-capable models.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func AnalyzeImage(ctx context.Context, imagePath string) (string, error) {
    // Read image file
    imageData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", fmt.Errorf("read image: %w", err)
    }

    // Create multimodal message
    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart("What do you see in this image? Describe it in detail."),
            schema.ImagePart(imageData, "image/jpeg"),
        ),
    }

    // Use vision-capable model
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-4o"), // Vision-capable model
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    factory := llms.NewFactory()
    llm, err := factory.CreateLLM("openai", config)
    if err != nil {
        return "", err
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", fmt.Errorf("generate: %w", err)
    }

    return response.Content, nil
}

func main() {
    ctx := context.Background()

    description, err := AnalyzeImage(ctx, "photo.jpg")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Image description: %s\n", description)
}
```

### Image URLs

Use image URLs instead of embedding raw bytes.

```go
func AnalyzeImageURL(ctx context.Context, llm llms.LLM, imageURL string) (string, error) {
    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart("Describe this image."),
            schema.ImageURLPart(imageURL),
        ),
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}
```

### Multiple Images

Analyze multiple images in a single request.

```go
func CompareImages(ctx context.Context, llm llms.LLM, image1Path, image2Path string) (string, error) {
    img1, _ := os.ReadFile(image1Path)
    img2, _ := os.ReadFile(image2Path)

    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart("Compare these two images. What are the differences?"),
            schema.ImagePart(img1, "image/jpeg"),
            schema.ImagePart(img2, "image/jpeg"),
        ),
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}
```

## Document Intelligence

Extract structured data from documents using vision.

```go
type Receipt struct {
    Vendor      string  `json:"vendor"`
    Date        string  `json:"date"`
    Total       float64 `json:"total"`
    Items       []Item  `json:"items"`
    PaymentMethod string `json:"payment_method"`
}

type Item struct {
    Name     string  `json:"name"`
    Quantity int     `json:"quantity"`
    Price    float64 `json:"price"`
}

func ExtractReceipt(ctx context.Context, llm llms.LLM, receiptImage []byte) (*Receipt, error) {
    // Define extraction schema
    schemaJSON, _ := GenerateSchema[Receipt]()

    messages := []schema.Message{
        schema.NewSystemMessage("You are an expert at extracting structured data from receipts."),
        schema.NewUserMessage(
            schema.TextPart("Extract all information from this receipt."),
            schema.ImagePart(receiptImage, "image/jpeg"),
        ),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
        llms.WithTemperature(0.0),
    )
    if err != nil {
        return nil, err
    }

    var receipt Receipt
    if err := json.Unmarshal([]byte(response.Content), &receipt); err != nil {
        return nil, fmt.Errorf("unmarshal: %w", err)
    }

    return &receipt, nil
}
```

### ID Card Verification

Extract and verify information from ID cards.

```go
type IDCard struct {
    Type         string `json:"type"` // "passport", "drivers_license", etc.
    FullName     string `json:"full_name"`
    DateOfBirth  string `json:"date_of_birth"`
    IDNumber     string `json:"id_number"`
    ExpiryDate   string `json:"expiry_date"`
    IssuingAuthority string `json:"issuing_authority"`
    Photo        bool   `json:"photo_present"`
}

func VerifyID(ctx context.Context, llm llms.LLM, idImage []byte) (*IDCard, bool, error) {
    schemaJSON, _ := GenerateSchema[IDCard]()

    messages := []schema.Message{
        schema.NewSystemMessage("Extract ID card information. Verify the document appears authentic."),
        schema.NewUserMessage(
            schema.TextPart("Extract information and check for signs of tampering."),
            schema.ImagePart(idImage, "image/jpeg"),
        ),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
    )
    if err != nil {
        return nil, false, err
    }

    var idCard IDCard
    if err := json.Unmarshal([]byte(response.Content), &idCard); err != nil {
        return nil, false, err
    }

    // Ask about authenticity
    verifyMessages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart("Does this ID appear authentic? Answer YES or NO with brief reasoning."),
            schema.ImagePart(idImage, "image/jpeg"),
        ),
    }

    verifyResponse, err := llm.Generate(ctx, verifyMessages)
    if err != nil {
        return &idCard, false, err
    }

    isAuthentic := strings.Contains(strings.ToUpper(verifyResponse.Content), "YES")

    return &idCard, isAuthentic, nil
}
```

## Audio Processing

### Audio Transcription with Analysis

Process audio beyond simple transcription.

```go
func AnalyzeAudio(ctx context.Context, llm llms.LLM, audioPath string) (string, error) {
    audioData, err := os.ReadFile(audioPath)
    if err != nil {
        return "", err
    }

    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart("Summarize this audio recording. Include: main topics, sentiment, and key decisions."),
            schema.AudioPart(audioData, "audio/mp3"),
        ),
    }

    // Use audio-capable model (e.g., Gemini 1.5)
    config := llms.NewConfig(
        llms.WithProvider("google"),
        llms.WithModelName("gemini-1.5-pro"),
        llms.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
    )

    factory := llms.NewFactory()
    audioLLM, err := factory.CreateLLM("google", config)
    if err != nil {
        return "", err
    }

    response, err := audioLLM.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}
```

### Meeting Intelligence

Extract structured insights from meetings.

```go
type MeetingSummary struct {
    Title           string   `json:"title"`
    Date            string   `json:"date"`
    Attendees       []string `json:"attendees"`
    Topics          []string `json:"topics"`
    Decisions       []string `json:"decisions"`
    ActionItems     []ActionItem `json:"action_items"`
    NextMeetingDate string   `json:"next_meeting_date"`
}

type ActionItem struct {
    Task       string `json:"task"`
    Assignee   string `json:"assignee"`
    Deadline   string `json:"deadline"`
}

func SummarizeMeeting(ctx context.Context, llm llms.LLM, audioData []byte) (*MeetingSummary, error) {
    schemaJSON, _ := GenerateSchema[MeetingSummary]()

    messages := []schema.Message{
        schema.NewSystemMessage("You are an expert meeting assistant. Extract structured information from meeting recordings."),
        schema.NewUserMessage(
            schema.TextPart("Analyze this meeting recording and extract all information."),
            schema.AudioPart(audioData, "audio/mp3"),
        ),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
    )
    if err != nil {
        return nil, err
    }

    var summary MeetingSummary
    if err := json.Unmarshal([]byte(response.Content), &summary); err != nil {
        return nil, err
    }

    return &summary, nil
}
```

## Combining Multiple Modalities

Process text, images, and audio together.

```go
func AnalyzeVideoFrame(ctx context.Context, llm llms.LLM, frame []byte, audioSegment []byte, timestamp time.Duration) (string, error) {
    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart(fmt.Sprintf("What's happening at %s in this video? Consider both visual and audio.", timestamp)),
            schema.ImagePart(frame, "image/jpeg"),
            schema.AudioPart(audioSegment, "audio/mp3"),
        ),
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}
```

## Multimodal RAG

Build RAG systems that index images and audio.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/vectorstore"
)

type MultimodalDocument struct {
    ID          string
    Type        string // "text", "image", "audio"
    Content     []byte
    TextSummary string // LLM-generated description
    Embedding   []float32
    Metadata    map[string]interface{}
}

func IndexMultimodalDocument(
    ctx context.Context,
    llm llms.LLM,
    embedder embeddings.Embedder,
    vectorDB vectorstore.VectorStore,
    doc MultimodalDocument,
) error {
    // Generate text summary for non-text content
    if doc.Type != "text" {
        summary, err := generateSummary(ctx, llm, doc)
        if err != nil {
            return fmt.Errorf("generate summary: %w", err)
        }
        doc.TextSummary = summary
    }

    // Embed the summary
    embedding, err := embedder.EmbedText(ctx, doc.TextSummary)
    if err != nil {
        return fmt.Errorf("embed: %w", err)
    }
    doc.Embedding = embedding

    // Store in vector database
    schemaDoc := schema.Document{
        PageContent: doc.TextSummary,
        Metadata: map[string]interface{}{
            "id":   doc.ID,
            "type": doc.Type,
            "original_content": doc.Content, // Or store URL
        },
    }

    if err := vectorDB.AddDocuments(ctx, []schema.Document{schemaDoc}, [][]float32{embedding}); err != nil {
        return fmt.Errorf("store: %w", err)
    }

    return nil
}

func generateSummary(ctx context.Context, llm llms.LLM, doc MultimodalDocument) (string, error) {
    var messages []schema.Message

    switch doc.Type {
    case "image":
        messages = []schema.Message{
            schema.NewUserMessage(
                schema.TextPart("Describe this image in detail for search indexing."),
                schema.ImagePart(doc.Content, "image/jpeg"),
            ),
        }

    case "audio":
        messages = []schema.Message{
            schema.NewUserMessage(
                schema.TextPart("Summarize this audio for search indexing."),
                schema.AudioPart(doc.Content, "audio/mp3"),
            ),
        }

    default:
        return "", fmt.Errorf("unsupported type: %s", doc.Type)
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}

// Search across multimodal documents
func SearchMultimodal(
    ctx context.Context,
    embedder embeddings.Embedder,
    vectorDB vectorstore.VectorStore,
    query string,
) ([]MultimodalDocument, error) {
    // Embed query
    queryEmbedding, err := embedder.EmbedText(ctx, query)
    if err != nil {
        return nil, err
    }

    // Search vector database
    results, err := vectorDB.SimilaritySearch(ctx, queryEmbedding, 10)
    if err != nil {
        return nil, err
    }

    // Convert results
    var docs []MultimodalDocument
    for _, result := range results {
        doc := MultimodalDocument{
            ID:          result.Metadata["id"].(string),
            Type:        result.Metadata["type"].(string),
            TextSummary: result.PageContent,
            // Retrieve original content from storage
        }
        docs = append(docs, doc)
    }

    return docs, nil
}
```

## Provider-Specific Features

### OpenAI GPT-4o

Best for high-quality image analysis with fine details.

```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4o"),
    llms.WithExtraOptions(map[string]interface{}{
        "detail": "high", // "low", "high", or "auto"
    }),
)
```

### Anthropic Claude 3

Excellent for document understanding and complex reasoning.

```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-5-sonnet-20241022"),
    llms.WithMaxTokens(4096), // Claude supports long outputs
)
```

### Google Gemini

Best for audio processing and long context.

```go
config := llms.NewConfig(
    llms.WithProvider("google"),
    llms.WithModelName("gemini-1.5-pro"),
    llms.WithExtraOptions(map[string]interface{}{
        "safety_settings": []map[string]string{
            {"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_ONLY_HIGH"},
        },
    }),
)
```

## Error Handling

Handle multimodal-specific errors.

```go
func AnalyzeImageWithFallback(ctx context.Context, imagePath string) (string, error) {
    imageData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", err
    }

    // Try primary provider
    result, err := tryProvider(ctx, "openai", "gpt-4o", imageData)
    if err == nil {
        return result, nil
    }

    log.Printf("Primary provider failed: %v, trying fallback", err)

    // Fallback to secondary provider
    result, err = tryProvider(ctx, "anthropic", "claude-3-5-sonnet-20241022", imageData)
    if err != nil {
        return "", fmt.Errorf("all providers failed: %w", err)
    }

    return result, nil
}

func tryProvider(ctx context.Context, provider, model string, imageData []byte) (string, error) {
    config := llms.NewConfig(
        llms.WithProvider(provider),
        llms.WithModelName(model),
    )

    factory := llms.NewFactory()
    llm, err := factory.CreateLLM(provider, config)
    if err != nil {
        return "", err
    }

    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart("Describe this image."),
            schema.ImagePart(imageData, "image/jpeg"),
        ),
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}
```

## Production Best Practices

When using multimodal AI in production:

1. **Optimize image sizes** - resize images to reduce costs and latency
2. **Use URLs when possible** - avoid base64 encoding overhead
3. **Implement rate limiting** - multimodal calls are expensive
4. **Cache results** - identical images/audio get identical descriptions
5. **Monitor costs** - multimodal tokens cost more than text
6. **Handle file formats** - convert to supported formats (JPEG, PNG, MP3)
7. **Validate inputs** - check file sizes and formats before API calls
8. **Use appropriate detail levels** - "low" for thumbnails, "high" for OCR
9. **Implement timeouts** - large files take longer to process
10. **Test provider capabilities** - not all providers support all modalities

## Next Steps

Now that you understand multimodal AI:
- Learn about [Document Processing](/guides/document-processing) for OCR pipelines
- Explore [Voice AI Pipeline](/guides/voice-ai) for real-time audio
- Read [RAG Recipes](/cookbook/rag-recipes) for multimodal search
- Check out [Multimodal Recipes](/cookbook/multimodal-recipes) for advanced patterns
