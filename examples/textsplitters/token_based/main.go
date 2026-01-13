// Example: Token-based text splitting
//
// This example demonstrates how to use custom length functions
// for token-based splitting, which is useful when working with
// LLM tokenizers (e.g., tiktoken for OpenAI models).
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

func main() {
	ctx := context.Background()

	// Sample text to split
	longText := `This is a long document that needs to be split based on token count rather than character count.
Token-based splitting is important when working with LLM APIs that have token limits.
For example, OpenAI's GPT models have context windows measured in tokens, not characters.
A simple character-based split might create chunks that exceed token limits when encoded.`

	fmt.Println("=== Token-based Splitting Example ===")
	fmt.Println()

	// Example 1: Simple token approximation (words * 1.3)
	// In production, you would use a real tokenizer like tiktoken
	fmt.Println("Example 1: Word-based token approximation")
	fmt.Println("----------------------------------------")

	// Approximate tokens as: number of words * 1.3 (rough estimate)
	tokenLengthFn := func(text string) int {
		words := strings.Fields(text)
		// Rough approximation: 1 word ≈ 1.3 tokens
		return int(float64(len(words)) * 1.3)
	}

	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(50),    // 50 tokens
		textsplitters.WithRecursiveChunkOverlap(10), // 10 tokens overlap
		textsplitters.WithRecursiveLengthFunction(tokenLengthFn),
	)
	if err != nil {
		log.Fatalf("Failed to create splitter: %v", err)
	}

	chunks, err := splitter.SplitText(ctx, longText)
	if err != nil {
		log.Fatalf("Failed to split text: %v", err)
	}

	fmt.Printf("Split text into %d chunks (target: 50 tokens each):\n", len(chunks))
	for i, chunk := range chunks {
		estimatedTokens := tokenLengthFn(chunk)
		fmt.Printf("  Chunk %d (~%d tokens): %.80s...\n", i+1, estimatedTokens, chunk)
	}

	// Example 2: Character-based splitting (default)
	fmt.Println("\nExample 2: Character-based splitting (for comparison)")
	fmt.Println("-----------------------------------------------------")

	charSplitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(200),   // 200 characters
		textsplitters.WithRecursiveChunkOverlap(40), // 40 characters overlap
		// No custom length function = uses character count
	)
	if err != nil {
		log.Fatalf("Failed to create splitter: %v", err)
	}

	charChunks, err := charSplitter.SplitText(ctx, longText)
	if err != nil {
		log.Fatalf("Failed to split text: %v", err)
	}

	fmt.Printf("Split text into %d chunks (target: 200 characters each):\n", len(charChunks))
	for i, chunk := range charChunks {
		fmt.Printf("  Chunk %d (%d chars): %.80s...\n", i+1, len(chunk), chunk)
	}

	// Example 3: Integration with real tokenizer (pseudo-code)
	fmt.Println("\nExample 3: Integration pattern for real tokenizers")
	fmt.Println("---------------------------------------------------")
	fmt.Println(`
// In production, you would integrate with a real tokenizer:
//
// import "github.com/tiktoken-go/tokenizer"
//
// enc, _ := tokenizer.Get(tokenizer.Cl100kBase)
// tokenLengthFn := func(text string) int {
//     tokens, _ := enc.Encode(text, nil, nil)
//     return len(tokens)
// }
//
// splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
//     textsplitters.WithRecursiveChunkSize(4096), // GPT-4 context window
//     textsplitters.WithRecursiveChunkOverlap(200),
//     textsplitters.WithRecursiveLengthFunction(tokenLengthFn),
// )
`)

	fmt.Println("\n✨ Token-based splitting example completed!")
	fmt.Println("\nNote: This example uses a simple word-based approximation.")
	fmt.Println("For production use, integrate with a real tokenizer library.")
}
