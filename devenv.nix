{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  # https://devenv.sh/packages/
  packages = [
    pkgs.golangci-lint
    pkgs.goreleaser
  ];

  # https://devenv.sh/languages/
  languages.go = {
    enable = true;
    version = "1.26.3";
  };

  # https://devenv.sh/basics/
  enterShell = ''
        echo "mdformat environment loaded"
        echo "================================"
        echo "  go              : $(go version)"
        echo "================================"
        echo "  golangci-lint   : $(golangci-lint --version)"
        echo "================================"
  '';

  # https://devenv.sh/tests/
  enterTest = ''
    echo "Running tests"
    git --version | grep --color=auto "${pkgs.git.version}"
  '';

  # https://devenv.sh/git-hooks/
  git-hooks.hooks = {
    # Lint shell scripts
    shellcheck.enable = true;

    # git checks
    commitizen.enable = true;
    check-merge-conflicts.enable = true;
    gitlint.enable = true;
    forbid-new-submodules.enable = true;

    # checks
    check-json.enable = true;
    check-yaml.enable = true;
    check-added-large-files.enable = true;
    check-executables-have-shebangs.enable = true;
    check-shebang-scripts-are-executable.enable = true;
    check-symlinks.enable = true;

    # fixers
    end-of-file-fixer.enable = true;
    fix-byte-order-marker.enable = true;

    fmt = {
      enable = true;
      name = "go fmt";
      entry = "make fmt";
      language = "system";
      pass_filenames = false;
    };
    vet = {
      enable = true;
      name = "go vet";
      entry = "make vet";
      language = "system";
      pass_filenames = false;
    };
    lint = {
      enable = true;
      name = "go lint";
      entry = "make lint";
      language = "system";
      pass_filenames = false;
    };
    test = {
      enable = true;
      name = "go test";
      entry = "make test";
      language = "system";
      pass_filenames = false;
    };
  };

  # See full reference at https://devenv.sh/reference/options/
}
