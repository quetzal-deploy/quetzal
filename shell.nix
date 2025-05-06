{
  nixpkgs ? import ./nixpkgs.nix,
  pkgs ? import nixpkgs { },
}:

let
  quetzal = pkgs.callPackage ./default.nix { };

in
pkgs.mkShell { inputsFrom = [ quetzal ]; }
