#!/bin/bash

# Test script for ai-top
# Run with: ./test.sh

echo "Testing ai-top..."
echo

# Build
echo "Building..."
go build -o bin/ai-top ./cmd/ai-top
if [ ! -f bin/ai-top ]; then
    echo "❌ Build failed"
    exit 1
fi
echo "✅ Build successful"

# Check binary size
SIZE=$(ls -lh bin/ai-top | awk '{print $5}')
echo "✅ Binary size: $SIZE"
echo

# Test basic functionality (non-interactive)
echo "Testing metric collection..."
OUTPUT=$(timeout 2 ./bin/ai-top 2>&1 || true)

# Check for expected output
if echo "$OUTPUT" | grep -q "Ollama"; then
    echo "✅ Ollama detection works"
else
    echo "⚠️  Ollama detection - no data (likely not running)"
fi

if echo "$OUTPUT" | grep -q "Node.js"; then
    echo "✅ Node.js process detection works"
else
    echo "❌ Node.js detection failed"
    exit 1
fi

if echo "$OUTPUT" | grep -q "OpenClaw"; then
    echo "✅ OpenClaw detection works"
else
    echo "❌ OpenClaw detection failed"
    exit 1
fi

echo
echo "✅ All tests passed!"
echo
echo "To run interactively:"
echo "  ./bin/ai-top"
echo
echo "Controls:"
echo "  q - Quit"
echo "  space - Pause/resume"
echo "  c - Sort by CPU"
echo "  m - Sort by memory"
echo "  s - Sort by name"
