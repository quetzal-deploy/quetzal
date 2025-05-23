{
  config,
  lib,
  pkgs,
  ...
}:

with lib;
with lib.types;

let

  ownerOptionsType = submodule (_: {
    options = {
      group = mkOption {
        type = str;
        description = "Group that will own the secret.";
        default = "root";
      };

      user = mkOption {
        type = str;
        description = "User who will own the secret.";
        default = "root";
      };
    };
  });

  keyOptionsType = submodule (_: {
    options = {
      destination = mkOption {
        type = str;
        description = "Remote path";
      };

      source = mkOption {
        type = str;
        description = "Local path";
      };

      owner = mkOption {
        default = { };
        type = ownerOptionsType;
        description = ''
          Owner of the secret.
        '';
      };

      permissions = mkOption {
        default = "0400";
        type = str;
        description = "Permissions expressed as octal.";
      };

      action = mkOption {
        default = [ ];
        type = listOf str;
        description = "Action to perform on remote host after uploading secret.";
      };

      mkDirs = mkOption {
        default = true;
        type = bool;
        description = ''
          Whether to create parent directories to secret destination.
          In particular, Quetzal will execute `sudo mkdir -p -m 755 /path/to/secret/destination`
          prior to moving the secret in place.
        '';
      };

      uploadAt = mkOption {
        default = "pre-activation";
        type = enum [
          "pre-activation"
          "post-activation"
        ];
        description = ''
          When to upload the secret.

          `pre-activation` (the default) will upload the secret and run any associated action prior to activating the system configuration.
          `post-activation` will upload the secret and run any associated action after activating the system configuration.
        '';
      };
    };
  });

  healthCheckType = submodule (_: {
    options = {
      cmd = mkOption {
        type = listOf cmdHealthCheckType;
        default = [ ];
        description = "List of command health checks";
      };
      http = mkOption {
        type = listOf httpHealthCheckType;
        default = [ ];
        description = "List of HTTP health checks";
      };
    };
  });

  httpHealthCheckType = types.submodule (_: {
    options = {
      description = mkOption {
        type = str;
        description = "Health check description";
      };
      host = mkOption {
        type = nullOr str;
        description = "Host name";
        default = null;
        #default = config.networking.hostName;
      };
      scheme = mkOption {
        type = str;
        description = "Scheme";
        default = "http";
      };
      port = mkOption {
        type = int;
        description = "Port number";
      };
      path = mkOption {
        type = path;
        description = "HTTP request path";
        default = "/";
      };
      headers = mkOption {
        type = attrsOf str;
        description = "HTTP request headers";
        default = { };
      };
      period = mkOption {
        type = int;
        description = "Seconds between checks";
        default = 2;
      };
      timeout = mkOption {
        type = int;
        description = "Timeout in seconds";
        default = 5;
      };
      insecureSSL = mkOption {
        type = bool;
        description = "Ignore SSL errors";
        default = false;
      };
    };
  });

  cmdHealthCheckType = types.submodule (_: {
    options = {
      description = mkOption {
        type = str;
        description = "Health check description";
      };
      cmd = mkOption {
        type = nullOr (listOf str);
        description = "Command to run as list";
        default = null;
      };
      period = mkOption {
        type = int;
        description = "Seconds between checks";
        default = 2;
      };
      timeout = mkOption {
        type = int;
        description = "Timeout in seconds";
        default = 5;
      };
    };
  });

in
{
  options.deployment = {

    targetHost = mkOption {
      type = str;
      default = "";
      description = ''
        The remote host used for deployment. If this is not set it will fallback to the deployments attribute name.
      '';
    };

    targetPort = mkOption {
      type = nullOr int;
      default = null;
      description = ''
        The port number of remote host used for deployment.
      '';
    };

    targetUser = mkOption {
      type = str;
      default = "";
      description = ''
        The remote user used for deployment. If this is not set it will fallback to the user specified in the
        <literal>SSH_USER</literal> environment variable or use the current local user as a last resort.
      '';
    };

    buildOnly = mkOption {
      type = bool;
      default = false;
      description = ''
        Set to true if the host will not be real or reachable.
        This is useful for system configs used to build iso's, local testing etc.
        Will make the following features unavailable for the host:
          push, deploy, check-health, upload-secrets, exec
      '';
    };

    substituteOnDestination = mkOption {
      type = bool;
      default = false;
      description = ''
        Sets the `--substitute-on-destination` flag on nix copy,
        allowing for the deployment target to use substitutes.
        See `nix copy --help`.
      '';
    };

    secrets = mkOption {
      default = { };
      example = {
        "nix-cache-signing-key" = {
          source = "../secrets/very-secret.txt";
          destination = "/var/secrets/very-secret.txt";
          owner.user = "nginx";
          owner.group = "root";
          permissions = "0400"; # this is the default
          action = [
            "sudo"
            "systemctl"
            "reload"
            "nginx.service"
          ]; # restart nginx after uploading the secret
        };
      };
      type = attrsOf keyOptionsType;
      description = ''
        Attrset where each attribute describes a key to be copied via ssh
        instead of through the Nix closure (keeping it out of the Nix store.)
      '';
    };

    healthChecks = mkOption {
      type = healthCheckType;
      description = ''
        Health check configuration.
      '';
      default = { };
    };

    preDeployChecks = mkOption {
      type = healthCheckType;
      description = ''
        Pre-check configuration.
      '';
      default = { };
    };
    tags = mkOption {
      type = listOf str;
      default = [ ];
      description = ''
        Host tags.
      '';
    };
  };

  # Creates a txt-file that lists all system healthcheck commands
  # The file will end up linked in /run/current-system along with
  # all derived dependencies.
  config.system.extraDependencies =
    let
      cmds = concatMap (h: h.cmd) (
        config.deployment.preDeployChecks.cmd ++ config.deployment.healthChecks.cmd
      );
    in
    [ (pkgs.writeText "healthcheck-commands.txt" (concatStringsSep "\n" cmds)) ];
}
