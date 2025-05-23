{ nixosTest, packages, ... }:

nixosTest {
  name = "quetzal-deployment-test";
  nodes =
    let
      boot.loader = {
        systemd-boot.enable = true;
        efi.canTouchEfiVariables = true;
      };
      services.openssh = {
        enable = true;
        startWhenNeeded = false;
      };
    in
    {
      deployer = _: {
        inherit services boot;
        environment.systemPackages = [ packages.quetzal ];
      };
      target = _: { inherit services boot; };
    };
  testScript = ''
    start_all()

    deployer.wait_for_unit("sshd")
    target.wait_for_unit("sshd")
  '';
}
