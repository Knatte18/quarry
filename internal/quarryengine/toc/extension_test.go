// extension_test.go table-drives LanguageForExtension over the extension set plus its edge cases,
// and pins ExtensionLanguages/ExtensionsForLanguage against the same map.

package toc

import (
	"reflect"
	"testing"
)

func TestLanguageForExtension(t *testing.T) {
	tests := []struct {
		name   string
		ext    string
		want   string
		wantOK bool
	}{
		{name: "go", ext: ".go", want: "go", wantOK: true},
		{name: "unknown", ext: ".java", want: "", wantOK: false},
		{name: "dotless", ext: "go", want: "go", wantOK: true},
		{name: "uppercase", ext: ".GO", want: "go", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := LanguageForExtension(tt.ext)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("LanguageForExtension(%q) = (%q, %v); want (%q, %v)", tt.ext, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestExtensionLanguages(t *testing.T) {
	want := []string{"go"}
	got := ExtensionLanguages()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtensionLanguages() = %v; want %v", got, want)
	}
}

func TestExtensionsForLanguage(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want []string
	}{
		{name: "go has one extension", lang: "go", want: []string{".go"}},
		{name: "unknown language returns nil", lang: "cobol", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtensionsForLanguage(tt.lang)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtensionsForLanguage(%q) = %v; want %v", tt.lang, got, tt.want)
			}
		})
	}
}
