{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  # https://devenv.sh/packages/
  packages = [
    pkgs.gnumake
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

    # Run each Go hook through `devenv shell` so the nix-provided toolchain
    # (go, golangci-lint, make) is on PATH even when the hook is invoked from a
    # shell where direnv/devenv is not active. Commands delegate to the Makefile
    # so the hooks and `make` stay in sync.
    fmt = {
      enable = true;
      name = "go fmt";
      entry = "devenv shell -- make fmt";
      language = "system";
      pass_filenames = false;
    };
    vet = {
      enable = true;
      name = "go vet";
      entry = "devenv shell -- make vet";
      language = "system";
      pass_filenames = false;
    };
    lint = {
      enable = true;
      name = "go lint";
      entry = "devenv shell -- make lint";
      language = "system";
      pass_filenames = false;
    };
    test = {
      enable = true;
      name = "go test";
      entry = "devenv shell -- make test";
      language = "system";
      pass_filenames = false;
    };
    test-integration = {
      enable = true;
      name = "golden-file integration tests";
      entry = "devenv shell -- make test-integration";
      language = "system";
      pass_filenames = false;
    };
  };

  # See full reference at https://devenv.sh/reference/options/
}
