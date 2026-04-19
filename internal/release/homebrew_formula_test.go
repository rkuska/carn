package release

import "testing"

func TestRenderHomebrewSourceFormula(t *testing.T) {
	t.Parallel()

	formula, err := RenderHomebrewSourceFormula(HomebrewSourceFormula{
		URL:    "https://github.com/rkuska/carn/archive/refs/tags/v1.2.3.tar.gz",
		SHA256: "abc123",
	})
	if err != nil {
		t.Fatalf("RenderHomebrewSourceFormula returned error: %v", err)
	}

	const want = `class Carn < Formula
  desc "Terminal UI for browsing local Claude and Codex session archives"
  homepage "https://github.com/rkuska/carn"
  url "https://github.com/rkuska/carn/archive/refs/tags/v1.2.3.tar.gz"
  sha256 "abc123"
  license "GPL-3.0-only"

  depends_on "go" => :build

  def install
    ldflags = [
      "-s -w",
      "-X github.com/rkuska/carn/internal/app.version=#{version}",
      "-X github.com/rkuska/carn/internal/app.commit=homebrew",
    ].join(" ")

    system "go", "build", *std_go_args(ldflags: ldflags), "."
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/carn --version")
  end
end
`

	if formula != want {
		t.Fatalf("unexpected formula output:\n%s", formula)
	}
}

func TestRenderHomebrewSourceFormulaMissingURL(t *testing.T) {
	t.Parallel()

	_, err := RenderHomebrewSourceFormula(HomebrewSourceFormula{SHA256: "abc123"})
	if err == nil {
		t.Fatal("expected error for missing url")
	}
	if err.Error() != "missing url" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderHomebrewSourceFormulaMissingSHA256(t *testing.T) {
	t.Parallel()

	_, err := RenderHomebrewSourceFormula(HomebrewSourceFormula{
		URL: "https://github.com/rkuska/carn/archive/refs/tags/v1.2.3.tar.gz",
	})
	if err == nil {
		t.Fatal("expected error for missing sha256")
	}
	if err.Error() != "missing sha256" {
		t.Fatalf("unexpected error: %v", err)
	}
}
