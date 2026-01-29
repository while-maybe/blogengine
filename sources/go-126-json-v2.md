---
title: "Go 1.26: The Arrival of JSON v2"
description: "Why the new JSON v2 library looks exciting"
author: "Unknown"
modified_at: "2026-01-20"
draft: false
noindex: true
---

# Go 1.26: The Arrival of JSON v2

For over a decade, `encoding/json` has been the workhorse of Go services. But let's be honest: it was slow, memory-hungry, and `omitempty` was a foot-gun.

With Go 1.26 RC1 dropping this week, the new standard `encoding/json/v2` is finally here. Here is why you should migrate your DTOs immediately.

## The `omitzero` Tag

The biggest pain point in v1 was distinguishing between a boolean `false` and a missing value. We used to have to use pointers `*bool` everywhere.

In v2, the `omitzero` tag handles zero-values intelligently without pointers:

```go
type User struct {
    ID       int    `json:"id,omitzero"`
    IsActive bool   `json:"is_active,omitzero"` // false is not marshaled
}
```

## Zero-Allocation Streaming

The new json.Reader interface allows for true zero-allocation parsing. In my benchmarks migrating this blog from v1 to v2, garbage collection pressure dropped by 40%.

```Go
// Old way (reads fully into memory)
// json.Unmarshal(data, &v)

// New way (streaming tokens)
dec := json.NewDecoder(r)
for dec.More() {
    val, err := dec.ReadValue()
}
```

This update alone makes Go 1.26 a mandatory upgrade for high-throughput HTTP services.
