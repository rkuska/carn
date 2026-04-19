package release

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"
)

type HomebrewSourceFormula struct {
	URL    string
	SHA256 string
}

var homebrewSourceFormulaTemplate = template.Must(template.New("homebrewSourceFormula").Parse(`class Carn < Formula
  desc "Terminal UI for browsing local Claude and Codex session archives"
  homepage "https://github.com/rkuska/carn"
  url "{{ .URL }}"
  sha256 "{{ .SHA256 }}"
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
`))

func RenderHomebrewSourceFormula(formula HomebrewSourceFormula) (string, error) {
	if formula.URL == "" {
		return "", errors.New("missing url")
	}
	if formula.SHA256 == "" {
		return "", errors.New("missing sha256")
	}

	var buffer bytes.Buffer
	if err := homebrewSourceFormulaTemplate.Execute(&buffer, formula); err != nil {
		return "", fmt.Errorf("renderHomebrewSourceFormula_execute: %w", err)
	}

	return buffer.String(), nil
}
