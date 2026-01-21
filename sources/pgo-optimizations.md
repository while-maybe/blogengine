# Profile-Guided Optimization (PGO) is now Default

Startups often ignore PGO because it sounds complex. "I need to collect a CPU profile, commit it to git, and build with flags?"

As of Go 1.26, PGO is effectively "Zero Config".

## The Auto-PGO Workflow

The `go build` command now automatically looks for a `default.pgo` profile in your main package directory.

If you are running a service in Kubernetes:

1. Curl the pprof endpoint: `curl -o default.pgo http://pod-ip/debug/pprof/profile?seconds=30`
2. Commit the file.
3. Deploy.

## The Result

I enabled this on my `mdtohtml` engine.

- **Build Time:** increased by 2s.
- **Runtime CPU:** decreased by 8%.
- **Inline Budget:** The compiler was able to inline my hot Markdown parsing functions that were previously considered "too complex."

If you aren't committing a `.pgo` file in 2026, you are leaving free performance on the table.
