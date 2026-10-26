{ pkgs, ... }:

{
  # Go and the tools the .githooks/pre-commit hook calls. The hook enters
  # this shell by itself when devenv is installed.
  languages.go = {
    enable = true;
    package = pkgs.go_1_27;
  };

  packages = [
    pkgs.golangci-lint
    pkgs.gopls
  ];

  # devenv puts GOPATH under .devenv inside the checkout, where
  # `-coverpkg=./...` counts the dependencies in its module cache as this
  # module's code and the coverage gate fails. Use Go's default instead.
  enterShell = ''
    export GOPATH="$HOME/go"
  '';
}
