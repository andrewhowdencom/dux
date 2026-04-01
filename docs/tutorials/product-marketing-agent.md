# Building a Product Marketing Agent

This tutorial details how to leverage Dux as a **Product Marketing Agent** to automatically draft release announcements, blog posts, and customer success case studies.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed.
- Core config pointing to a high-quality language model suited for creative writing.

## Step 1: Agent Definition

In your `agents.yaml`, establish the marketing persona:

### YAML Configuration Example

```yaml
- name: "product-marketer"
  provider: "ollama-local"
  context:
    system: |
      You are an elite Product Marketing Manager.
      Write compelling, benefit-driven copy. Focus on the "Why" and the value generated
      for the end user, rather than just technical specifications.
```

### Go Library Example

You can trigger this agent programmatically within an automated build/release pipeline or a custom Go app using the Dux Adapter API:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
)

engine := adapter.New(
	adapter.WithProvider(prv), 
	adapter.WithHistory(history.NewInMemory()),
	adapter.WithSystemPrompt("You are an elite Product Marketing Manager mapping technical outputs to compelling copy..."),
)
```

## Step 2: Ideation and Drafting

Launch the agent with:

```bash
dux chat --agent product-marketer
```

Provide the agent with a list of technical JIRA tickets or commit messages, and it will generate a customer-facing release drafted in your specified marketing tone.

## Current Limitations & Known Gaps

To fully operate as an automated marketing department, this agent requires capabilities Dux does not yet possess:

- **Brand Voice Tuning**: While system prompts help, achieving true "brand voice" usually requires extensive Few-Shot prompting, embeddings of past successful blog posts, or fine-tuning, none of which are seamlessly supported by Dux natively yet.
- **Multi-Modal Generation**: Marketing often requires generating or manipulating images alongside text. Dux is strictly text-based and cannot interface with image generation providers (e.g., Midjourney, DALL-E) or output rich media.
- **Document Ingestion (RESOLVED)**: We can now resolve URL scraping, API fetching, or basic PDF reading natively by writing modular CLI tools or programmatic go functions and mapping them directly to the `adapter.Engine`'s ToolResolver!
