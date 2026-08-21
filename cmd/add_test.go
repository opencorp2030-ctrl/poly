package cmd

import (
	"reflect"
	"testing"
)

func TestAddOrReplaceSpec(t *testing.T) {
	got := addOrReplaceSpec([]string{"tap:ripgrep@15.1.0", "npm:lodash@4.0.0"}, "lodash", "npm:lodash@4.17.21")
	want := []string{"tap:ripgrep@15.1.0", "npm:lodash@4.17.21"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("replace: got %v, want %v", got, want)
	}

	got = addOrReplaceSpec([]string{"tap:ripgrep@15.1.0"}, "jq", "brew:jq@1.8.2")
	want = []string{"tap:ripgrep@15.1.0", "brew:jq@1.8.2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("append: got %v, want %v", got, want)
	}

	// Existing entries for a different adapter of the same name are replaced too.
	got = addOrReplaceSpec([]string{"pip:requests@2.31.0"}, "requests", "npm:requests@2.32.0")
	want = []string{"npm:requests@2.32.0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cross-adapter replace: got %v, want %v", got, want)
	}
}
