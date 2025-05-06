{
  nixpkgs ? import ./nixpkgs.nix,
  pkgs ? import nixpkgs { },
  version ? "dev",
}:

pkgs.buildGoModule rec {
  name = "quetzal-unstable-${version}";
  inherit version;

  src = pkgs.nix-gitignore.gitignoreSource [ ] ./.;

  ldflags = [
    "-X main.version=${version}"
    "-X main.assetRoot=${placeholder "lib"}"
  ];

  nativeBuildInputs = [ pkgs.installShellFiles ];

  vendorHash = "sha256-/530PgUjiJm9snAcC8Db4vEU2dnhmH/9xkp6mFo/ngM=";

  postInstall = ''
    mkdir -p $lib
    cp -v ./data/*.nix $lib
    installShellCompletion --cmd quetzal \
      --bash <($out/bin/quetzal --completion-script-bash) \
      --zsh <($out/bin/quetzal --completion-script-zsh)
  '';

  outputs = [
    "out"
    "lib"
  ];

  meta = {
    homepage = "https://github.com/quetzal-deploy/deploy";
    description = "Quetzal is a NixOS host manager written in Golang.";
    mainProgram = "quetzal";
  };
}
