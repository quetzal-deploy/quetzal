{
  description = "NixOS deployment tool, supporting parallel and custom deployments";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
  };

  nixConfig = {
    experimental-features = [ "pipe-operator" ];
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.treefmt-nix.flakeModule
      ];

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        {
          config,
          self',
          inputs',
          pkgs,
          system,
          lib,
          ...
        }:
        {
          packages.default = pkgs.buildGoModule rec {
            pname = "quetzal";
            version = "0.1";
            src = pkgs.nix-gitignore.gitignoreSource [ ] ./.;

            vendorHash = "sha256-pprnK2JKmPuR3Q+F8+vMDEdowlb3oX4BOOzW8NGOqgs=";

            ldflags = [
              "-X main.assets=${placeholder "assets"}"
              "-X main.version=${version}"
            ];

            postInstall = ''
              mkdir -p $assets
            '';

            outputs = [
              "out"
              "assets"
            ];
          };

          devShells.default = pkgs.mkShell {
            inputsFrom = [ self'.packages.default ];

            packages = with pkgs; [
              gotools
            ];
          };

          treefmt = {
            programs = {
              gofmt.enable = true;
              goimports.enable = true;
              nixfmt.enable = true;
            };

            settings = {
              formatter.goimports = {
                includes = [ "*.go" ];
                options = [
                  "-w"
                  "-local"
                  "github.com/quetzal-deploy"
                ];
              };
            };
          };
        };
      flake = {

      };
    };
}
