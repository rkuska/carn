{ pkgs, ... }:

{
  # Go and the tools the .githooks/pre-commit hook calls, for machines
  # without them installed: `devenv shell -- git commit`.
  languages.go = {
    enable = true;
    package = pkgs.go_1_27;
  };

  packages = [
    pkgs.golangci-lint
    pkgs.gopls
  ];
}
