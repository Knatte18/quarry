# Status

```yaml
phase: approved-lsp-mirrored-tools
slug: quarry-mcp-wrapper
branch: quarry-mcp-wrapper
plan: _mill/plan
parent: main
module_verify_baseline: clean
task: Add an MCP wrapper for quarry
task_description: |
  Add an MCP wrapper for quarry
```

## Timeline

```text
discussing  '2026-08-29T05:55:57Z'
discussion-fix-r6  '2026-08-29T06:46:05Z'
discussed  '2026-08-29T06:46:05Z'
planning  '2026-08-29T07:01:08Z'
plan-review-r1  '2026-08-29T07:07:15Z'
plan-fix-r1  '2026-08-29T07:09:21Z'
plan-review-r2  '2026-08-29T07:15:05Z'
plan-fix-r2  '2026-08-29T07:16:48Z'
plan-review-r3  '2026-08-29T07:22:17Z'
plan-fix-r3  '2026-08-29T07:23:32Z'
plan-review-r4  '2026-08-29T07:30:50Z'
plan-fix-r4  '2026-08-29T07:31:55Z'
plan-review-r5  '2026-08-29T07:37:29Z'
plan-fix-r5  '2026-08-29T07:38:49Z'
planned  '2026-08-29T07:38:58Z'
implementing  '2026-08-29T07:39:30Z'
approved-export-cli-helpers  '2026-08-29T07:44:35Z'
approved-mcpserver-foundation  '2026-08-29T07:53:18Z'
approved-lsp-mirrored-tools  '2026-08-29T08:01:36Z'
```

## Batches

```yaml
batches:
  - name: export-cli-helpers
    state: approved
    implementer_session: 75982287-6523-41a9-98ac-05d429040078
    start_sha: 304f170e41b1ed0c82d64b0ba4e4c5872466ffe9
    commit_sha: 010187c71bb72765c1f47c59be7dfa7c17470694
    verify_baseline_failures: []
  - name: mcpserver-foundation
    state: approved
    implementer_session: 193e3330-3228-4f1f-a4b6-75ccee006bd9
    start_sha: 79b53a2ea766abb0256a0e463ef565005bb9c5c8
    commit_sha: 9129a49c1849fc66a94878556ecdab39d894cdb1
    verify_baseline_failures: ["FAIL\t./internal/mcpserver/... [setup failed]"]
  - name: lsp-mirrored-tools
    state: approved
    implementer_session: 0a5ccb39-795c-4965-ad8a-d107a8345a95
    start_sha: 7c02e1b3c5f1498adab9b1babec414c3f2acc688
    commit_sha: 14a32eb0bcc16fe817f925d6bf19c09a3f2df087
    verify_baseline_failures: ["FAIL\t./internal/mcpserver/... [setup failed]"]
  - name: quarry-native-lsp-tools
    state: running
    implementer_session: b33b3db3-3a54-4d5d-91af-ec8f0e7f12e5
    start_sha: defdb58b5eed4ac336d94ea23c17bab5090074c0
    verify_baseline_failures: ["FAIL\t./internal/mcpserver/... [setup failed]"]
  - name: toc-tools
    state: pending
    verify_baseline_failures: ["FAIL\t./internal/mcpserver/... [setup failed]"]
  - name: server-binary-and-transport-tests
    state: pending
    verify_baseline_failures: ["FAIL\t./internal/mcpserver/... [setup failed]"]
  - name: integration-and-docs
    state: pending
    verify_baseline_failures: ["FAIL\t./internal/mcpserver/... [setup failed]"]
```
