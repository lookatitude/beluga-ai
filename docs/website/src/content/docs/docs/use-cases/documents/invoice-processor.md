---
title: AI Invoice Processing Pipeline
description: "Process thousands of invoices daily with LLM-powered extraction, validation, approval routing, and accounting integration."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AI invoice processing, invoice extraction, automated accounts payable, document workflow, fintech AI, Beluga AI, Go, OCR invoices"
---

Financial services companies process thousands of invoices daily, each requiring data extraction, validation, approval routing, and accounting system integration. Manual processing is not just slow — it introduces systematic errors: misread amounts delay payments, missed due dates incur penalties, and inconsistent approval routing creates compliance gaps. When invoice volume spikes at month-end or quarter-close, the manual process becomes a bottleneck that directly impacts cash flow and vendor relationships.

The challenge goes beyond OCR and extraction. Invoices arrive in varied formats from hundreds of vendors, each with different layouts. Extracted data must be validated against business rules (line items must sum to total, dates must be valid, vendor must exist in the system). Approval routing depends on amount thresholds and business unit, and the entire process must be auditable for compliance.

Automated invoice processing using Beluga AI's orchestration capabilities chains these stages into a single resilient workflow with checkpointing, so no invoice is lost even if the system restarts mid-processing.

## Solution Architecture

Beluga AI's `orchestration/` package coordinates multi-stage workflows with checkpointing for recovery. The pipeline uses a sequential workflow (`NewSequentialWorkflow`) where each stage is a `core.Runnable` — parse, extract, validate, approve, integrate. This design makes each stage independently testable and replaceable. Checkpointing saves state between stages so a failure in approval routing doesn't require re-extracting data from the PDF.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Invoice PDF  │───▶│ Parse &      │───▶│ LLM          │
│ (Various     │    │ OCR          │    │ Extraction   │
│  Formats)    │    └──────────────┘    └──────┬───────┘
└──────────────┘                                │
                                                ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Accounting   │◀───│ Approval     │◀───│ Validation   │
│ System       │    │ Workflow     │    │ (Rules)      │
└──────────────┘    └──────────────┘    └──────────────┘
        ▲
        │
┌───────┴──────┐
│ Checkpoints  │
│ (Recovery)   │
└──────────────┘
```

## Workflow Implementation

Define the invoice processing workflow with multiple stages:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/orchestration"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// InvoiceProcessor orchestrates invoice processing workflow
type InvoiceProcessor struct {
    workflow orchestration.Workflow
    model    llm.ChatModel
}

func NewInvoiceProcessor(ctx context.Context) (*InvoiceProcessor, error) {
    // Create LLM for data extraction
    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })
    if err != nil {
        return nil, err
    }

    // Create workflow with stages
    workflow := orchestration.NewSequentialWorkflow(
        "invoice-processing",
        []core.Runnable{
            NewParseStep(),
            NewExtractionStep(model),
            NewValidationStep(),
            NewApprovalStep(),
            NewIntegrationStep(),
        },
    )

    return &InvoiceProcessor{
        workflow: workflow,
        model:    model,
    }, nil
}

// ProcessInvoice processes an invoice through the workflow
func (p *InvoiceProcessor) ProcessInvoice(ctx context.Context, invoiceData []byte) (*ProcessedInvoice, error) {
    input := map[string]interface{}{
        "invoice_data": invoiceData,
    }

    result, err := p.workflow.Invoke(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("workflow failed: %w", err)
    }

    return result.(*ProcessedInvoice), nil
}
```

## Data Extraction with LLM

Traditional OCR extracts text but doesn't understand document structure — it can't distinguish a vendor name from an address, or a line item total from the invoice total. LLM-based extraction understands document semantics and maps them to structured fields. Using `llm.WithResponseFormat("json_object")` constrains the response to valid JSON, ensuring the output can be reliably parsed into the `InvoiceData` struct:

```go
type InvoiceData struct {
    InvoiceNumber string    `json:"invoice_number"`
    Date          string    `json:"date"`
    Vendor        string    `json:"vendor"`
    Total         float64   `json:"total"`
    LineItems     []LineItem `json:"line_items"`
}

type LineItem struct {
    Description string  `json:"description"`
    Quantity    int     `json:"quantity"`
    UnitPrice   float64 `json:"unit_price"`
    Total       float64 `json:"total"`
}

// ExtractionStep extracts invoice data using LLM
type ExtractionStep struct {
    model llm.ChatModel
}

func NewExtractionStep(model llm.ChatModel) core.Runnable {
    return &ExtractionStep{model: model}
}

func (s *ExtractionStep) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
    data := input.(map[string]interface{})
    invoiceText := data["parsed_text"].(string)

    // Build extraction prompt
    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{
                Text: `Extract invoice data into JSON format with these fields:
- invoice_number: string
- date: string (YYYY-MM-DD)
- vendor: string
- total: number
- line_items: array of {description, quantity, unit_price, total}

Return only valid JSON.`,
            },
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: invoiceText},
        }},
    }

    // Use structured output for guaranteed JSON
    resp, err := s.model.Generate(ctx, msgs,
        llm.WithResponseFormat("json_object"),
    )
    if err != nil {
        return nil, fmt.Errorf("extraction failed: %w", err)
    }

    // Parse JSON response
    var invoiceData InvoiceData
    if err := json.Unmarshal([]byte(resp.Parts[0].(schema.TextPart).Text), &invoiceData); err != nil {
        return nil, fmt.Errorf("invalid JSON: %w", err)
    }

    data["invoice_data"] = invoiceData
    return data, nil
}
```

## Validation Step

Validate extracted data against business rules:

```go
// ValidationStep validates invoice data
type ValidationStep struct{}

func NewValidationStep() core.Runnable {
    return &ValidationStep{}
}

func (s *ValidationStep) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
    data := input.(map[string]interface{})
    invoice := data["invoice_data"].(InvoiceData)

    var errors []string

    // Validate required fields
    if invoice.InvoiceNumber == "" {
        errors = append(errors, "missing invoice number")
    }
    if invoice.Vendor == "" {
        errors = append(errors, "missing vendor")
    }
    if invoice.Total <= 0 {
        errors = append(errors, "invalid total amount")
    }

    // Validate line item totals match invoice total
    lineItemTotal := 0.0
    for _, item := range invoice.LineItems {
        lineItemTotal += item.Total
    }
    if math.Abs(lineItemTotal-invoice.Total) > 0.01 {
        errors = append(errors, fmt.Sprintf("line items total %.2f does not match invoice total %.2f", lineItemTotal, invoice.Total))
    }

    // Validate date format
    if _, err := time.Parse("2006-01-02", invoice.Date); err != nil {
        errors = append(errors, "invalid date format")
    }

    if len(errors) > 0 {
        return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
    }

    data["validated"] = true
    return data, nil
}
```

## Approval Workflow

Route invoices through approval based on amount thresholds:

```go
// ApprovalStep routes invoice for approval
type ApprovalStep struct {
    approvalService ApprovalService
}

func NewApprovalStep() core.Runnable {
    return &ApprovalStep{
        approvalService: NewApprovalService(),
    }
}

func (s *ApprovalStep) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
    data := input.(map[string]interface{})
    invoice := data["invoice_data"].(InvoiceData)

    // Determine approval route based on amount
    var approvers []string
    switch {
    case invoice.Total < 1000:
        approvers = []string{"manager@company.com"}
    case invoice.Total < 10000:
        approvers = []string{"manager@company.com", "director@company.com"}
    default:
        approvers = []string{"manager@company.com", "director@company.com", "cfo@company.com"}
    }

    // Request approval
    approved, err := s.approvalService.RequestApproval(ctx, invoice, approvers)
    if err != nil {
        return nil, fmt.Errorf("approval request failed: %w", err)
    }

    if !approved {
        return nil, fmt.Errorf("invoice rejected")
    }

    data["approved"] = true
    return data, nil
}
```

## Production Considerations

### Checkpointing for Recovery

Implement checkpointing to recover from failures:

```go
// ProcessWithCheckpointing processes with automatic recovery
func (p *InvoiceProcessor) ProcessWithCheckpointing(ctx context.Context, invoiceID string, invoiceData []byte) (*ProcessedInvoice, error) {
    checkpoint := NewCheckpointManager()

    // Check for existing checkpoint
    if state, err := checkpoint.Load(ctx, invoiceID); err == nil {
        logger.Info("resuming from checkpoint", "invoice_id", invoiceID, "stage", state.Stage)
        return p.resumeFromCheckpoint(ctx, state)
    }

    // Process with checkpointing
    input := map[string]interface{}{
        "invoice_id":   invoiceID,
        "invoice_data": invoiceData,
    }

    // Wrap each stage to checkpoint after completion
    for i, stage := range p.workflow.Stages {
        result, err := stage.Invoke(ctx, input)
        if err != nil {
            // Save checkpoint before failing
            checkpoint.Save(ctx, invoiceID, CheckpointState{
                Stage: i,
                Data:  input,
            })
            return nil, err
        }
        input = result.(map[string]interface{})
    }

    // Clear checkpoint on success
    checkpoint.Delete(ctx, invoiceID)

    return input["result"].(*ProcessedInvoice), nil
}
```

### Retry with Exponential Backoff

Retry failed stages with exponential backoff:

```go
import "github.com/lookatitude/beluga-ai/resilience"

func (s *ExtractionStep) InvokeWithRetry(ctx context.Context, input interface{}) (interface{}, error) {
    policy := resilience.RetryPolicy{
        MaxAttempts:    3,
        InitialBackoff: 1 * time.Second,
        MaxBackoff:     10 * time.Second,
        BackoffFactor:  2.0,
        Jitter:         true,
    }

    return resilience.Retry(ctx, policy, func(ctx context.Context) (interface{}, error) {
        return s.Invoke(ctx, input)
    })
}
```

### Batch Processing

Process multiple invoices in parallel:

```go
// ProcessBatch processes multiple invoices concurrently
func (p *InvoiceProcessor) ProcessBatch(ctx context.Context, invoices []Invoice) ([]ProcessedInvoice, error) {
    results := make([]ProcessedInvoice, len(invoices))
    errors := make([]error, len(invoices))

    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 10)  // Limit concurrency to 10

    for i, invoice := range invoices {
        wg.Add(1)
        go func(idx int, inv Invoice) {
            defer wg.Done()

            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            result, err := p.ProcessInvoice(ctx, inv.Data)
            results[idx] = *result
            errors[idx] = err
        }(i, invoice)
    }

    wg.Wait()

    // Check for any errors
    for _, err := range errors {
        if err != nil {
            return results, fmt.Errorf("batch processing had errors: %w", err)
        }
    }

    return results, nil
}
```

## Related Resources

- [Multi-Stage ETL](/use-cases/multi-stage-etl/) for complex pipeline patterns
- [Orchestration Guide](/guides/orchestration/) for workflow design
- [Resilience Patterns](/guides/resilience/) for retry and circuit breaker setup
