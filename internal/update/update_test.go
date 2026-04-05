package update

import "testing"

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "newer patch", left: "v0.1.2", right: "v0.1.1", want: 1},
		{name: "older patch", left: "v0.1.1", right: "v0.1.2", want: -1},
		{name: "same version", left: "v0.1.2", right: "v0.1.2", want: 0},
		{name: "handles prerelease suffix", left: "v0.1.2-beta.1", right: "v0.1.1", want: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compareVersions(tt.left, tt.right)
			if got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestAssetNameFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		goos   string
		goarch string
		want   string
		ok     bool
	}{
		{goos: "linux", goarch: "amd64", want: "kernus-linux-amd64", ok: true},
		{goos: "darwin", goarch: "arm64", want: "kernus-darwin-arm64", ok: true},
		{goos: "windows", goarch: "amd64", want: "kernus-windows-amd64.exe", ok: true},
		{goos: "windows", goarch: "arm64", ok: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.goos+"-"+tt.goarch, func(t *testing.T) {
			t.Parallel()
			got, err := assetNameFor(tt.goos, tt.goarch)
			if tt.ok {
				if err != nil {
					t.Fatalf("assetNameFor(%q, %q) returned error: %v", tt.goos, tt.goarch, err)
				}
				if got != tt.want {
					t.Fatalf("assetNameFor(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("assetNameFor(%q, %q) = %q, want error", tt.goos, tt.goarch, got)
			}
		})
	}
}
