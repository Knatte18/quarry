Uses:
- internal/fabriccli#addWeftVerbs
- internal/fabricengine#Fabric.MergeInProgress
- internal/fabricengine#Fabric.MergeContinue
- internal/fabricengine#mergeInProgressReason
- internal/fabricengine#Fabric.mergeRecordExists
- internal/fabricengine#Fabric.foreignMergeStatePresent
- internal/fabricengine#ErrMergeInProgress
- internal/gitrepo#Repo.MergeHeadPresent
- internal/mergeresolve#Resolver.Resolve

<!-- KICKSTART-PACK:BEGIN -->
internal/fabriccli#addWeftVerbs → internal/fabriccli/weft_verbs.go 39-355
    func addWeftVerbs(cmd *cobra.Command)
internal/fabricengine#Fabric.MergeInProgress → internal/fabricengine/mergelifecycle.go 414-420
    func (f *Fabric) MergeInProgress() (bool, error)
internal/fabricengine#Fabric.MergeContinue → internal/fabricengine/mergelifecycle.go 248-348
    func (f *Fabric) MergeContinue(msg string) (res MergeResult, err error)
internal/fabricengine#mergeInProgressReason → internal/fabricengine/mergeguards.go 159-170
    func mergeInProgressReason(f *Fabric) ([]string, error)
internal/fabricengine#Fabric.mergeRecordExists → internal/fabricengine/mergestate.go 177-185
    func (f *Fabric) mergeRecordExists() (bool, error)
internal/fabricengine#Fabric.foreignMergeStatePresent → internal/fabricengine/mergestate.go 250-291
    func (f *Fabric) foreignMergeStatePresent() (bool, error)
internal/fabricengine#ErrMergeInProgress → internal/fabricengine/mergeerrors.go 148-150
    type ErrMergeInProgress struct
internal/gitrepo#Repo.MergeHeadPresent → internal/gitrepo/merge.go 249-262
    func (r *Repo) MergeHeadPresent() (bool, error)
internal/mergeresolve#Resolver.Resolve → internal/mergeresolve/mergeresolve.go 21-61
    func (r *Resolver) Resolve(ctx context.Context, source string) (Result, error)
<!-- KICKSTART-PACK:END -->

Read all listed spans in parallel, in one turn, before doing anything else.
