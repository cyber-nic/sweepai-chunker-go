# Go SweepAI Chunking

This repository provides a **Go-based** implementation of the [Sweep AI chunking algorithm](https://docs.sweep.dev/blogs/chunking-2m-files), which is further refined in [this follow-up blog post](https://docs.sweep.dev/blogs/chunking-improvements). The original algorithm chunks large code files by parsing their Abstract Syntax Trees (ASTs), then splits them into byte/line ranges for more efficient LLM processing. It **significantly improves** upon naive line-based splitting by preserving semantic boundaries and reducing context window overflow.

## Overview

- **AST-based chunking**: Uses Tree-sitter grammars (compiled into `.so` files) to parse source code.
- **Recursive splitting**: Recursively traverses AST nodes until each chunk is below a specified size limit.
- **Fallback**: For files that cannot be parsed by Tree-sitter (e.g., unknown languages), a naive line-based approach is used.

## Key Files

- **`chunker.go`**:

  - Implements the core chunking logic (similar to the Python SweepAI approach).

- **`cmd/main.go`**:
  - A minimal demo that uses the **chunker** to parse and chunk itself (`main.go`) as an example.

## Usage

1. **Clone this repository**

   ```bash
   git clone https://github.com/cyber-nic/sweepai-chunker-go
   ```

2. **Build and run the Go program**

   ```bash
   go build -o chunker .
   ./chunker
   ```

   It will process the current directory.

## Credits and Further Reading

- **Sweep AI Blogs**:
  - [Chunking 2m Files](https://docs.sweep.dev/blogs/chunking-2m-files)
  - [Chunking Improvements](https://docs.sweep.dev/blogs/chunking-improvements)

These posts detail the motivation behind AST-based chunking, performance benefits, and the original Python implementation. This Go version replicates those core ideas while leveraging **build-time** Tree-sitter compilation for cleaner, more reliable deployment.
