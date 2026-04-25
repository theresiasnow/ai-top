# Ollama Model Loading Strategy

## Overview

Your AI development environment uses 4 Ollama models totaling ~22GB of GPU VRAM:
- **qwen2.5-coder:14b** (8GB) - Coding and reasoning specialist
- **gemma4:latest** (9GB) - General-purpose reasoning
- **qwen3:8b** (5GB) - Medium model for quick tasks  
- **nomic-embed-text:latest** (<1GB) - Semantic embeddings

## Current Loading Strategy (Recommended)

### All Models Pre-loaded (Current Setup ✓)

**Configuration**: All 4 models loaded into GPU VRAM simultaneously
- Total memory: 22GB VRAM
- Ollama system RAM: ~37MB (models are GPU-resident)
- Status: ✅ Recommended for your use case

**Advantages**:
- ✅ Instant model switching - no wait time
- ✅ Seamless interactive development
- ✅ Best for coding workflows (qwen-coder always ready)
- ✅ Multiple models available for cross-validation
- ✅ No cold-start latency on first inference

**Disadvantages**:
- ✗ Requires 22GB GPU VRAM (high utilization)
- ✗ No VRAM available for other GPU tasks
- ✗ Less efficient for batch-only workloads

**Who should use this**: 
- Interactive AI development
- Coding assistance (primary use case)
- Multiple model comparison
- Real-time inference needs

## Alternative: On-Demand Loading

If you need to free up VRAM for other GPU tasks:

### Strategy 1: Keep Hot Models Only

**Configuration**: Load only critical models, demand-load others
```
Always Loaded:
  • qwen2.5-coder:14b (8GB) - coding
  • gemma4:latest (9GB) - general purpose
  
On-Demand Load:
  • qwen3:8b (5GB) - load when needed
  • nomic-embed-text (<1GB) - load for embeddings
```

**VRAM Impact**: 17GB always in use, 5GB freed for other tasks

**Trade-offs**:
- ✅ Frees 5GB for other ML workloads
- ✗ 2-5 second wait for qwen3:8b on first use
- ✗ Not ideal for rapid model switching

**Use cases**:
- Running another GPU model simultaneously
- Batch processing workflows
- Memory-constrained deployments

### Strategy 2: Hybrid (Balanced)

**Configuration**: Keep hot models, pre-warm the medium model
```
Always Loaded:
  • qwen2.5-coder:14b (8GB)
  • gemma4:latest (9GB)

Pre-warm (load on first use):
  • qwen3:8b (5GB)

Cold (load on demand):
  • nomic-embed-text (<1GB)
```

**VRAM Impact**: Up to 22GB when all models loaded

**Characteristics**:
- ✅ Fast for primary tasks (hot models instant)
- ✅ Reasonable load time for secondary tasks (~2s)
- ✅ Good balance of speed and resource efficiency
- ✓ Recommended if you sometimes need freed VRAM

## Model Priority Mapping

The ai-top monitor categorizes models by loading priority:

| Priority | Example | Behavior | VRAM Strategy |
|----------|---------|----------|---------------|
| 🔥 Hot | qwen2.5-coder, gemma4 | Always in GPU memory | Persistent |
| 🌡️ Warm | qwen3:8b | Pre-load on first use | Load once |
| ❄️ Cold | nomic-embed-text | Load when needed | On-demand |

## How to Implement On-Demand Loading

Ollama doesn't have built-in on-demand loading, but you can:

### Option A: Manual Model Management

Unload when not needed:
```bash
# Keep models in memory
ollama serve

# In another terminal, manually unload:
curl -X DELETE http://localhost:11434/api/model/qwen3:8b

# Reload when needed:
ollama run qwen3:8b
```

### Option B: Orchestration Script

Create a script to manage model memory:
```bash
#!/bin/bash
# Keep hot models, unload warm models after X minutes of inactivity

HOT_MODELS=("qwen2.5-coder:14b" "gemma4:latest")
WARM_MODELS=("qwen3:8b")
INACTIVITY_TIMEOUT=300  # 5 minutes

# Monitor and unload inactive warm models
for model in "${WARM_MODELS[@]}"; do
  # Check last access time
  # If inactive > timeout: curl -X DELETE http://localhost:11434/api/model/$model
done
```

### Option C: Load Balancing Service

Use a proxy between your app and Ollama:
- Ollama Load Balancer (experimental)
- Monitor which models are being used
- Automatically unload low-priority models

## Performance Comparison

| Strategy | Hot Ready | Warm Ready | Cold Ready | VRAM Used | Use Case |
|----------|-----------|-----------|-----------|-----------|----------|
| **All Pre-loaded** | Instant | Instant | Instant | 22GB | Interactive dev ✅ |
| **Hot Only** | Instant | 2-5s | 2-5s | 17GB | Memory-constrained |
| **Hybrid** | Instant | ~2s | ~2s | ≤22GB | Balanced |
| **On-demand** | ~2s | ~2s | ~2s | ~10GB | Batch processing |

## Monitoring with ai-top

The Ollama tab in ai-top shows:
- Current model load status
- Priority classification (🔥🌡️❄️)
- Memory footprint
- Suggested unload candidates (future feature)

## Recommendations

### ✅ For Your Use Case (Interactive AI Development)

**Stick with current: All models pre-loaded**

Why:
1. Coding workflows benefit from instant qwen-coder availability
2. You have sufficient VRAM (24GB+ GPU observed)
3. Model switching is seamless
4. No cold-start penalties for inference
5. Best user experience

### If You Need More VRAM

**Try Hybrid Strategy**:
1. Keep hot models always loaded (17GB)
2. Unload qwen3:8b when not actively coding
3. Reload as needed (2-5s wait, acceptable for batch work)
4. Frees 5GB for other GPU tasks

### For Production / Batch Processing

**Use On-Demand Strategy**:
1. Run only one model at a time
2. Load/unload based on job queue
3. Minimal VRAM footprint
4. Accept inference latency as trade-off

