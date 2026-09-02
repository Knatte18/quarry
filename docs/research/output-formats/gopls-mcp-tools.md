# gopls v0.23.0 — `gopls mcp` tools/list

```
go_diagnostics
    Provides Go workspace diagnostics.

Checks for parse and build errors across the entire Go workspace. If provided,
"files" holds absolute paths for active files, on which additional linting is
performed.

    input: files

go_file_context
    Summarizes a file's cross-file dependencies
    input: file

go_package_api
    Provides a summary of a Go package API
    input: packagePaths

go_rename_symbol
    Renames a symbol in the Go workspace

For example, given arguments {"file": "/path/to/foo.go", "symbol": "Foo", "new_name": "Bar"},
go_rename_symbol returns the edits necessary to rename the symbol "Foo" (located in the file foo.go) to
"Bar" across the Go workspace.
    input: file, new_name, symbol

go_search
    Search for symbols in the Go workspace.

Search for symbols using case-insensitive fuzzy search, which may match all or
part of the fully qualified symbol name. For example, the query 'foo' matches
Go symbols 'Foo', 'fooBar', 'futils.Oboe', 'github.com/foo/bar.Baz'.

Results are limited to 100 symbols.

    input: query

go_symbol_references
    Provides the locations of references to a (possibly qualified)
package-level Go symbol referenced from the current file.

For example, given arguments {"file": "/path/to/foo.go", "name": "Foo"},
go_symbol_references returns references to the symbol "Foo" declared
in the current package.

Similarly, given arguments {"file": "/path/to/foo.go", "name": "lib.Bar"},
go_symbol_references returns references to the symbol "Bar" in the imported lib
package.

Finally, symbol references supporting querying fields and methods: symbol
"T.M" selects the "M" field or method of the "T" type (or value), and "lib.T.M"
does the same for a symbol in the imported package "lib".

    input: file, symbol

go_vulncheck
    Runs a vulnerability check on the Go workspace.

	The check is performed on a given package pattern within a specified directory.
	If no directory is provided, it defaults to the workspace root.
	If no pattern is provided, it defaults to "./...".
    input: dir, pattern

go_workspace
    Summarize the Go programming language workspace
    input: 

```
