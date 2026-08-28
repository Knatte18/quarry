// client.ts is an impact fixture: it exists only to be a file whose extension resolves to a
// language with no registered toc Strategy, exercising the per-entry degradation path.

export function greet(name: string): string {
    return "hello, " + name;
}
