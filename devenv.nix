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

  # devenv puts GOPATH under .devenv, and Go makes its module cache
  # read-only, which stops `rm -rf` and `git worktree remove` on the
  # checkout.
  env.GOFLAGS = "-modcacherw";
}
