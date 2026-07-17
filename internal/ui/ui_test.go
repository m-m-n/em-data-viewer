package ui

import (
	"reflect"
	"testing"
)

// TestRunArgs_ExcludesConfigPathArgument references AC-1: the argument slice
// passed to the GTK application's Run call must never contain the
// application-specific config-path argument, regardless of how many
// arguments the process was invoked with.
func TestRunArgs_ExcludesConfigPathArgument(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "invoked with an explicit config path",
			args: []string{"data-viewer", "/custom/config.yaml"},
			want: []string{"data-viewer"},
		},
		{
			name: "invoked with no arguments",
			args: []string{"data-viewer"},
			want: []string{"data-viewer"},
		},
		{
			name: "invoked with extra stray arguments",
			args: []string{"data-viewer", "/custom/config.yaml", "extra"},
			want: []string{"data-viewer"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := osArgs
			osArgs = tc.args
			defer func() { osArgs = old }()

			got := runArgs()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("runArgs() = %v, want %v", got, tc.want)
			}
			for _, a := range got {
				if a == "/custom/config.yaml" {
					t.Errorf("runArgs() = %v, contains config-path argument", got)
				}
			}
		})
	}
}
