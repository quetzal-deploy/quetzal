let
  pkgs =
    import (builtins.fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixos-25.05.tar.gz")
      { };

  planner = import /home/adtu/src/quetzal/data/planner.nix;
  qz = planner.lib;

  kvmHardware =
    { lib, modulesPath, ... }:
    {
      imports = [ (modulesPath + "/profiles/qemu-guest.nix") ];

      boot = {
        initrd.availableKernelModules = [
          "ahci"
          "xhci_pci"
          "virtio_pci"
          "sr_mod"
          "virtio_blk"
        ];
        initrd.kernelModules = [ ];
        kernelModules = [ "kvm-intel" ];
        extraModulePackages = [ ];

        loader.systemd-boot.enable = true;
        loader.efi.canTouchEfiVariables = true;
      };

      fileSystems."/" = {
        label = "nixos";
        fsType = "ext4";
      };

      fileSystems."/boot" = {
        label = "boot";
        fsType = "vfat";
        options = [
          "fmask=0022"
          "dmask=0022"
        ];
      };

      networking.useDHCP = lib.mkDefault true;
      nixpkgs.hostPlatform = lib.mkDefault "x86_64-linux";
    };

  common =
    { lib, modulesPath, ... }:
    {
      system.stateVersion = "25.05";

      users.users.adtu = {
        isNormalUser = true;
        extraGroups = [ "wheel" ]; # Enable ‘sudo’ for the user.
        openssh.authorizedKeys.keys = [
          "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJcoBNuMsH3ukDcNfY1oggYgHeLULe7J8YLfHP97PXK+ adtu@dbc.dk"
        ];
      };

      users.users.adam = {
        isNormalUser = true;
        extraGroups = [ "wheel" ]; # Enable ‘sudo’ for the user.
        hashedPassword = "$y$j9T$z6sBxiuj.uF6Mi60pMELB/$Ky2E4I27MW8C7cVkgDlpqt.Gntfu4xGB/DJXt/UFhV2";
        openssh.authorizedKeys.keys = [
          "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIH75Lc2tcIMLxm45v32j9ihTKZqdu6+xvXUbbLWXWoaW mail@adamtulinius.dk"
        ];
      };

      nix.settings.trusted-users = [ "@wheel" ];

      services.openssh.enable = true;

      environment.systemPackages = with pkgs; [
        htop
        vim
        wget
      ];
    };

  nginx =
    { lib, modulesPath, ... }:
    {
      services.prometheus.enable = true;
      networking.firewall.allowedTCPPorts = [ 9090 ];

      deployment = {
        healthChecks = {
          http = [
            {
              scheme = "http";
              port = 9090;
              path = "/";
              description = "Check whether prometheus is running.";
              period = 1; # number of seconds between retries
            }
          ];
        };
      };
    };

  healthChecks =
    { lib, modulesPath, ... }:
    {
      deployment = {
        healthChecks = {
          cmd = [
            #            {
            #              cmd = [
            #                "true"
            #              ];
            #              description = "Testing that 'true' works.";
            #            }
          ];
        };
      };
    };

in
{
  network = {
    inherit pkgs;
    description = "simple hosts";
    ordering = {
      tags = [
        "db"
        "web"
      ];
    };

    plans = {
      sprint-bump =
        args:
        qz.mkDefaultDeployPlan (
          args
          // {
            action = "boot";
            reboot = true;
          }
        );
    };

    # ordering = { labels = [ "role=db" "role=web" ]; };
    # ordering = {
    #   labels = [
    #     { label = "kind"; value = "db"; }
    #     { label = "kind"; value = "web"; }
    #   ];
    # };
    constraints = [
      {
        selector = {
          label = "_";
          value = "host";
        };
        maxUnavailable = 3;
      }
      {
        selector = {
          label = "location";
          value = "*";
        };
        maxUnavailable = 1000;
      }
      {
        selector = {
          label = "type";
          value = "*";
        };
        maxUnavailable = 1;
      }
    ];
  };

  "kvm01" = _: {
    deployment = {
      labels = {
        type = "web";
        location = "dc1";
      };
      targetHost = "192.168.122.184";
    };
    imports = [
      common
      kvmHardware
      healthChecks
      nginx
    ];
  };

  "kvm02" = _: {
    deployment = {
      labels = {
        type = "db";
        location = "dc1";
      };
      targetHost = "192.168.122.164";
    };
    imports = [
      common
      kvmHardware
      healthChecks
      nginx
    ];
  };

  "kvm03" = _: {
    deployment = {
      tags = [ "web" ];
      labels = {
        type = "web";
        location = "dc2";
      };
      targetHost = "192.168.122.185";
    };
    imports = [
      common
      kvmHardware
      healthChecks
      nginx
    ];
  };

  "kvm04" = _: {
    deployment = {
      tags = [ "web" ];
      labels = {
        type = "db";
        location = "dc2";
      };
      targetHost = "192.168.122.127";
    };
    imports = [
      common
      kvmHardware
      healthChecks
      nginx
    ];
  };

  "kvm05" = _: {
    deployment = {
      tags = [ "web" ];
      labels = {
        type = "web";
        location = "dc3";
      };
      targetHost = "192.168.122.122";
    };
    imports = [
      common
      kvmHardware
      healthChecks
      nginx
    ];
  };

  "kvm06" = _: {
    deployment = {
      tags = [ "web" ];
      labels = {
        type = "db";
        location = "dc3";
      };
      targetHost = "192.168.122.116";
    };
    imports = [
      common
      kvmHardware
      healthChecks
      nginx
    ];
  };
}
