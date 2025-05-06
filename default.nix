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

  vendorHash = "sha256-Mi0SdvmYao6rLt8+bFcUv2AjHkJTLP85zGka1/cCPzQ=";

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
