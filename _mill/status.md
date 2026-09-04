# Status

```yaml
phase: pr-pending
slug: facade-cli-toc
branch: facade-cli-toc
plan: _mill/plan
parent: main
module_verify_baseline: clean
task: Facade + CLI, toc (T5a)
task_description: |
  Facade + CLI, toc (T5a)
```

## Timeline

```text
discussing  '2026-09-04T05:34:45Z'
discussion-fix-r1  '2026-09-04T05:56:58Z'
discussion-fix-r2  '2026-09-04T06:04:01Z'
discussion-fix-r3  '2026-09-04T06:09:40Z'
discussed  '2026-09-04T06:09:40Z'
planning  '2026-09-04T06:20:05Z'
plan-review-r1  '2026-09-04T06:25:38Z'
plan-fix-r1  '2026-09-04T06:29:29Z'
plan-review-r2  '2026-09-04T06:34:52Z'
plan-fix-r2  '2026-09-04T06:37:29Z'
plan-review-r3  '2026-09-04T06:44:09Z'
plan-fix-r3  '2026-09-04T06:45:39Z'
planned  '2026-09-04T06:45:49Z'
implementing  '2026-09-04T06:46:39Z'
approved-facade-core  '2026-09-04T06:51:25Z'
approved-facade-renderers  '2026-09-04T07:01:24Z'
approved-cli-parsing  '2026-09-04T07:09:52Z'
approved-cli-pipeline  '2026-09-04T07:16:42Z'
approved-goldens-and-after  '2026-09-04T07:22:46Z'
holistic-reviewing  '2026-09-04T07:23:19Z'
holistic-approved  '2026-09-04T07:27:35Z'
done  '2026-09-04T07:28:18Z'
pr-pending  '2026-09-04T07:30:00Z'
```

## Batches

```yaml
batches:
  - name: facade-core
    state: approved
    implementer_session: 48778a4b-4ab9-4c53-a7a3-16943a7de8da
    start_sha: aaf1e2a31f03185093683b8888c280453aed5a47
    commit_sha: 7ab2727a3877abd088d0a70b67fd3f95c96e0c5f
    verify_baseline_failures: ["FAIL\t./quarry/... [setup failed]"]
  - name: facade-renderers
    state: approved
    implementer_session: 881ac749-a050-4b17-92a6-10c29eb84180
    start_sha: 3bf697114b8bedb0ab6d69792f17f27f35a6e02c
    commit_sha: 1875d2862e77a461d43a54bda5183e4002491c1a
    verify_baseline_failures: ["FAIL\t./quarry/... [setup failed]"]
  - name: cli-parsing
    state: approved
    implementer_session: 519ecd10-5815-46a9-8869-710d26d1a5f2
    start_sha: cbe52b295163bfe8ceaea0a8cfdc2eebbee3b658
    commit_sha: 1bc814bc75a9c407200e0073c9f8ba1cb3db54dd
    verify_baseline_failures: ["FAIL\t./internal/cli/... [setup failed]"]
  - name: cli-pipeline
    state: approved
    implementer_session: 543f3894-cd0b-4af7-ada0-ce114bee9636
    start_sha: e5457df015a5ad48523d9078f59639268f53c6ed
    commit_sha: 3061e6a1e9f91dba4b6ee83d19f24e378eea32b3
    verify_baseline_failures: ["FAIL\t./internal/cli/... [setup failed]", "FAIL\t./cmd/quarry/... [setup failed]"]
  - name: goldens-and-after
    state: approved
    implementer_session: 90342950-b8bb-4afc-9a9e-1acce809e46a
    start_sha: 5a97ff12b0ff68e7b916580729719e23372ac6cd
    commit_sha: b9ffd653872cb0a53005993fa7dca312a42bd615
    verify_baseline_failures: ["FAIL\t./internal/cli/... [setup failed]"]
```
