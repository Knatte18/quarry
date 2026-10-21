// csharp_type_test.go covers CanonicalCSharpType.
// It checks the alias table, the qualifier and whitespace rules, modifiers, tuples, and idempotence.

package glyph

import "testing"

func TestCanonicalCSharpType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Boolean", "Boolean", "bool"},
		{"BooleanQualified", "System.Boolean", "bool"},
		{"Byte", "Byte", "byte"},
		{"ByteQualified", "System.Byte", "byte"},
		{"SByte", "SByte", "sbyte"},
		{"SByteQualified", "System.SByte", "sbyte"},
		{"Char", "Char", "char"},
		{"CharQualified", "System.Char", "char"},
		{"Decimal", "Decimal", "decimal"},
		{"DecimalQualified", "System.Decimal", "decimal"},
		{"Double", "Double", "double"},
		{"DoubleQualified", "System.Double", "double"},
		{"Single", "Single", "float"},
		{"SingleQualified", "System.Single", "float"},
		{"Int16", "Int16", "short"},
		{"Int16Qualified", "System.Int16", "short"},
		{"UInt16", "UInt16", "ushort"},
		{"UInt16Qualified", "System.UInt16", "ushort"},
		{"Int32", "Int32", "int"},
		{"Int32Qualified", "System.Int32", "int"},
		{"UInt32", "UInt32", "uint"},
		{"UInt32Qualified", "System.UInt32", "uint"},
		{"Int64", "Int64", "long"},
		{"Int64Qualified", "System.Int64", "long"},
		{"UInt64", "UInt64", "ulong"},
		{"UInt64Qualified", "System.UInt64", "ulong"},
		{"Object", "Object", "object"},
		{"ObjectQualified", "System.Object", "object"},
		{"String", "String", "string"},
		{"StringQualified", "System.String", "string"},
		{"IntPtr", "IntPtr", "nint"},
		{"IntPtrQualified", "System.IntPtr", "nint"},
		{"UIntPtr", "UIntPtr", "nuint"},
		{"UIntPtrQualified", "System.UIntPtr", "nuint"},
		{"GenericArgument", "List<System.Int32>", "List<int>"},
		{"NestedGenericArguments", "Dictionary<String,List<Int64>>", "Dictionary<string,List<long>>"},
		{"ArrayElement", "Int32[]", "int[]"},
		{"NullableElement", "Int32?", "int?"},
		{"PointerElement", "Int32*", "int*"},
		{"GlobalAlias", "global::System.String", "string"},
		{"GlobalOtherType", "global::Foo.Bar", "Foo.Bar"},
		{"WhitespaceInGenerics", "List< int >", "List<int>"},
		{"WhitespaceAroundComma", "Dictionary< string , int >", "Dictionary<string,int>"},
		{"WhitespaceAroundDot", "System . Int32", "int"},
		{"ModifierRef", "ref  int", "ref int"},
		{"ModifierOut", "out int", "out int"},
		{"ModifierIn", "in int", "in int"},
		{"ModifierParams", "params   string[]", "params string[]"},
		{"ModifierThis", "this Box", "this Box"},
		{"ModifierScopedRef", "scoped ref Span<int>", "scoped ref Span<int>"},
		{"ModifierRefReadonly", "ref readonly Int32", "ref readonly int"},
		{"TupleElementNames", "(int a, string b)", "(int,string)"},
		{"TupleElementNamesWithAliases", "(Int32 a, String b)", "(int,string)"},
		{"TupleGenericElementName", "(List<int> a, int[] b)", "(List<int>,int[])"},
		{"TupleWithoutNames", "(int, string)", "(int,string)"},
		{"NestedTuple", "((int x, int y) point, string s)", "((int,int),string)"},
		{"NullableKept", "Box?", "Box?"},
		{"NullableTupleKept", "(int a, int b)?", "(int,int)?"},
		{"ArrayKept", "int[,]", "int[,]"},
		{"AliasSuffixIdentifierKept", "MyInt32", "MyInt32"},
		{"QualifiedAliasSuffixKept", "Foo.Int32", "Foo.Int32"},
		{"NonSystemQualifierKept", "Other.System.Int32", "Other.System.Int32"},
		{"OtherNamespaceQualifiedKept", "System.Text.Rune", "System.Text.Rune"},
		{"AliasQualifierKept", "alias::Foo", "alias::Foo"},
		{"VerbatimIdentifierKept", "@class", "@class"},
		{"Empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CanonicalCSharpType(tt.input)
			if got != tt.want {
				t.Errorf("CanonicalCSharpType(%q) = %q; want %q", tt.input, got, tt.want)
			}
			if again := CanonicalCSharpType(got); again != got {
				t.Errorf("CanonicalCSharpType(%q) = %q; want idempotent %q", got, again, got)
			}
		})
	}
}
