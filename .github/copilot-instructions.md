## Project baseline and build storage

Follow [project AGENTS.md](../AGENTS.md) for all changes in this repository:

1. **Technical baseline**: Read the relevant sections of [ARCHITECTURE.md](../ARCHITECTURE.md) and domain terms in [GLOSSARY.md](../GLOSSARY.md) before modifying core workflows, modules, DSH/Host integration, plugins/custom spaces, UI, performance, data, or release compatibility. Keep `C:/AI/Deepseek-Harness-Desktop` strictly read-only, preserve existing workspace edits and user data, and only advance migrations authorized by the current task.
2. **Build storage and disk constraints**: Before installing dependencies, copying runtimes, building, packaging, verifying, or cleaning outputs, follow the build-storage rules in [project AGENTS.md](../AGENTS.md) — reuse user-level shared caches, keep the 50 GiB C: free-space floor, clean creator-owned temporary artifacts in `finally`/`defer`/`t.Cleanup` without descending into junctions, and respect all protected-data boundaries.
