package sandbox

import "testing"

func TestSplitCollectionPathSupportsVersionedPrefixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		wantBase  string
		wantMoid  string
	}{
		{
			name:     "api version prefix",
			path:     "/api/v1/example/Widgets/widget-1",
			wantBase: "/api/v1/example/Widgets",
			wantMoid: "widget-1",
		},
		{
			name:     "version prefix only",
			path:     "/v2/example/widgets/widget-2",
			wantBase: "/v2/example/widgets",
			wantMoid: "widget-2",
		},
		{
			name:     "no version prefix",
			path:     "/example/widgets/widget-3",
			wantBase: "/example/widgets",
			wantMoid: "widget-3",
		},
		{
			name:     "single segment path",
			path:     "/widgets",
			wantBase: "/widgets",
			wantMoid: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotBase, gotMoid := splitCollectionPath(tt.path)
			if gotBase != tt.wantBase || gotMoid != tt.wantMoid {
				t.Fatalf("splitCollectionPath(%q) = (%q, %q), want (%q, %q)", tt.path, gotBase, gotMoid, tt.wantBase, tt.wantMoid)
			}
		})
	}
}
