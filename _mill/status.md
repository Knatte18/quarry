# Status

```yaml
phase: approved-python-csharp-strategies
slug: toc-verbs
branch: toc-verbs
plan: _mill/plan
parent: main
module_verify_baseline: clean
task: Add file/dir toc verbs (Tree-sitter-backed)
task_description: |
  Add file/dir toc verbs (Tree-sitter-backed)
```

## Timeline

```text
discussing  '2026-08-27T16:28:09Z'
discussed  '2026-08-27T18:19:39Z'
planning  '2026-08-27T19:11:30Z'
plan-review-r1  '2026-08-27T19:22:00Z'
plan-fix-r1  '2026-08-27T19:26:21Z'
plan-review-r2  '2026-08-28T05:26:23Z'
plan-fix-r2  '2026-08-28T05:29:55Z'
plan-review-r3  '2026-08-28T05:43:18Z'
plan-fix-r3  '2026-08-28T05:45:29Z'
plan-review-r4  '2026-08-28T05:51:09Z'
plan-fix-r4  '2026-08-28T05:53:30Z'
plan-review-r5  '2026-08-28T06:01:03Z'
plan-fix-r5  '2026-08-28T06:01:23Z'
plan-review-r6  '2026-08-28T06:09:19Z'
plan-fix-r6  '2026-08-28T06:10:55Z'
plan-review-r7  '2026-08-28T06:19:45Z'
planned  '2026-08-28T06:20:05Z'
implementing  '2026-08-28T06:20:35Z'
approved-treesitter-backend  '2026-08-28T06:27:37Z'
approved-toc-scaffolding  '2026-08-28T06:35:10Z'
approved-go-strategy  '2026-08-28T06:43:48Z'
approved-python-csharp-strategies  '2026-08-28T06:59:54Z'
```

## Batches

```yaml
batches:
  - name: treesitter-backend
    state: approved
    implementer_session: 3a5a1371-3269-4d83-9202-2feb12c1e3ac
    start_sha: 9c1f0f83a86ec5fbe598c5d4d927a0255fa266a9
    commit_sha: a7a3d9289201f3f4776fc433062e88927349cc49
    verify_baseline_failures: ["FAIL\t./internal/quarryengine/treesitter [setup failed]"]
  - name: toc-scaffolding
    state: approved
    implementer_session: 7ac2c536-be61-4a74-baec-c982562bbf8c
    start_sha: fe6384b3b08de73babe59df4c9db522591e6b068
    commit_sha: de6a2a868ee86789376e02410dae1e598210324d
    verify_baseline_failures: ["FAIL\t./internal/quarryengine/toc [setup failed]"]
  - name: go-strategy
    state: approved
    implementer_session: bd2e15ea-3887-4baa-983e-da32e4b9d7c9
    start_sha: f3bb6c89d34af5e96f96ea7c661fcc0710e73854
    commit_sha: 97aa6711114b5c31bf83e5d0c9557cc12b5238bc
    verify_baseline_failures: ["FAIL\t./internal/quarryengine/toc [setup failed]"]
  - name: python-csharp-strategies
    state: approved
    implementer_session: 35816808-26b0-4d25-bf8b-ab1e4c312b14
    start_sha: 06bce1aa4c422810cdc43c6534249d6b72512469
    commit_sha: b7ea9d0de331f27def742ed4b5803495a067337c
    verify_baseline_failures: ["FAIL\t./internal/quarryengine/toc [setup failed]"]
  - name: toc-entry-points
    state: pending
    verify_baseline_failures: ["FAIL\t./internal/quarryengine/toc [setup failed]"]
  - name: facade-and-cli
    state: pending
    verify_baseline_failures: ["FAIL\t./internal/quarryengine/toc [setup failed]"]
  - name: doc-sentences-config
    state: pending
    verify_baseline_failures: ["FAIL\t./internal/quarryengine/toc [setup failed]"]
  - name: docs-and-sweep
    state: pending
    verify_baseline_failures: []
```
