{ pkgs, ... }:
{
  projectRootFile = "flake.nix";
  programs = {
    nixfmt.enable = true; # nix formatter
    statix.enable = true; # nix static analysis
    shellcheck.enable = true; # bash/shell
    taplo.enable = true; # toml
    yamlfmt.enable = true; # yaml
    gofmt.enable = true;
    goimports.enable = true;
  };
  settings = {
    formatter = {
      nixfmt.includes = [
        "*.nix"
        "./data/*"
      ];
      statix.includes = [
        "*.nix"
        "./data/*"
      ];
      goimports.entry = "";
      goimports = {
        includes = [ "*.go" ];
        excludes = [ "vendor/*" ];
        options = [
          "-w"
          "-local"
          "github.com/quetzal-deploy"
        ];
      };
    };
  };
}
