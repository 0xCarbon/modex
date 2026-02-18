# Metadata
- **Source**: local
- **Issue ID**: MODEX-010
- **Repo Root**: .

# Problem
The current implementation of `prompts/coding_modern_go.md` is marked as a "failed" partial work in the git history (commit `15f5e96`). Although it contains substantial content (~990 lines), it only includes 22 modernizer analyzers in the "Complete Modernizer Table," whereas the project specification requires coverage of all 24 analyzers available in or planned for the Go 1.26 toolchain (as per the MODEX roadmap). Additionally, the prompt needs to be audited to ensure it perfectly aligns with the cratedex `coding_modern_rust.md` template structure, specifically regarding its prescription of the modex tool usage workflow and its handling of Go 1.25+ features like `testing/synctest` and `sync.WaitGroup.Go()`.

# Changes
- **prompts/coding_modern_go.md**:
    - Update the "Complete Modernizer Table" to include all 24 analyzers (identifying and adding the 2 missing ones).
    - Audit and refine the **Tool Usage Workflow** to ensure it explicitly directs the LLM to use `get_go_version_info`, `search_docs`, `get_diagnostics`, and `get_modernize_diagnostics` in the correct sequence.
    - Validate the **Concurrency Safety** section, ensuring it provides concrete before/after examples for modern patterns (e.g., `wg.Go()` vs `wg.Add(1); go ...`).
    - Ensure the **GODEBUG Awareness** section accurately describes the implications of version upgrades on runtime behavior.
    - Ensure the file remains substantial (400+ lines) while adhering to the 600-900 line target if possible (tightening redundant sections if needed).

# Verification
Run the following shell script to verify the prompt file's existence, length, and completeness of the modernizer table:
```bash
#!/bin/bash
set -e

FILE="prompts/coding_modern_go.md"

# 1. Verify file exists
if [ ! -f "$FILE" ]; then
    echo "Error: $FILE does not exist."
    exit 1
fi

# 2. Verify line count is substantial (> 400 lines)
LINE_COUNT=$(wc -l < "$FILE")
if [ "$LINE_COUNT" -lt 400 ]; then
    echo "Error: File is too short ($LINE_COUNT lines)."
    exit 1
fi

# 3. Verify the Modernizer Table contains exactly 24 analyzers
# Pattern matches rows like: | `name` | 1.XX | Replaces ... | Use ... |
ANALYZER_COUNT=$(grep -c "| `.*` | [0-9.]* | .* | .* |" "$FILE")
if [ "$ANALYZER_COUNT" -ne 24 ]; then
    echo "Error: Modernizer table has $ANALYZER_COUNT entries, expected 24."
    exit 1
fi

# 4. Verify presence of key sections
grep -q "## Tool Usage Workflow" "$FILE"
grep -q "## Concurrency Safety" "$FILE"
grep -q "## GODEBUG Awareness" "$FILE"

echo "Verification successful: $FILE is complete and well-structured."
```

# Out of Scope
- Implementation of the MCP server tools themselves (`get_diagnostics`, etc.).
- Automated injection of this prompt into LLM context windows (this issue covers only the content).
- Creation of other language-specific prompts.
