# Quetzal

[![Flake Checks](https://github.com/quetzal-deploy/quetzal/actions/workflows/checks.yml/badge.svg?branch=master)](https://github.com/quetzal-deploy/quetzal/actions/workflows/build.yaml)
[![Example Configs](https://github.com/quetzal-deploy/quetzal/actions/workflows/build-test.yaml/badge.svg?branch=master)](https://github.com/quetzal-deploy/quetzal/actions/workflows/build.yaml)

Quetzal is a tool for managing existing NixOS hosts - basically a fancy wrapper around `nix-build`, `nix copy`, `nix-env`, `/nix/store/.../bin/switch-to-configuration`, `scp` and more.
Quetzal supports updating multiple hosts in a row, and with support for health checks makes it fairly safe to do so.

The work that is the basis for Quetzal was originally started as a series of patches for https://github.com/DBCDK/morph, but since the changes are going to be substantial I'm not sure this will be recognizable as morph (and it might fail entirely), so for the time being this work will happen in this fork. That way there is also lower expectations of anything resembling backwards compatibility. The work exists in a very disorganized branch, `experimental`, that has served as a bit of a playground, but now the plan is to extract these features neatly into the master branch, little by little, so that work can yet again be based on the master branch.

Features migrated from `experimental` to `master`.
- [ ] event system
- [ ] execution engine / planner
- [ ] constraint system
- [ ] new UI
- [ ] daemon mode
- [ ] ...

Warning: Just because something exists in master, nothing will be considered stable until a first release.


# Old docs

## Notable features

* multi host support
* health checks
* no state


## Installation and prerequisites

Quetzal requires `nix` (at least v2), `ssh` and `scp` to be available on `$PATH`.
It should work on any modern Linux distribution, but NixOS is the only one we test on.

Pre-built binaries are not provided, since we install Quetzal through an overlay.

The easiest way to get Quetzal up and running is to fork this repository and run `nix-build`, which should result in a store path containing the Quetzal binary.
Consider checking out a specific tag, or at least pin the version of Quetzal you're using somehow.


## Using Quetzal

All commands support a `--help` flag; `quetzal --help` as of v1.0.0:
```
$ quetzal --help
usage: quetzal [<flags>] <command> [<args> ...]

NixOS host manager

Flags:
  --help     Show context-sensitive help (also try --help-long and --help-man).
  --version  Show application version.
  --dry-run  Don't do anything, just eval and print changes

Commands:
  help [<command>...]
    Show help.

  build [<flags>] <deployment>
    Evaluate and build deployment configuration to the local Nix store

  push [<flags>] <deployment>
    Build and transfer items from the local Nix store to target machines

  deploy [<flags>] <deployment> <switch-action>
    Build, push and activate new configuration on machines according to switch-action

  check-health [<flags>] <deployment>
    Run health checks

  upload-secrets [<flags>] <deployment>
    Upload secrets

  exec [<flags>] <deployment> <command>...
    Execute arbitrary commands on machines
```

Notably, `quetzal deploy` requires a `<switch-action>`.
The switch-action must be one of `dry-activate`, `test`, `switch` or `boot` corresponding to `nixos-rebuild` arguments of the same name.
Refer to the [NixOS manual](https://nixos.org/nixos/manual/index.html#sec-changing-config) for a detailed description of switch-actions.

For help on this and other commands, run `quetzal <cmd> --help`.

Example deployments can be found in the `examples` directory, and built as follows:
```
$ quetzal build examples/simple.nix
Selected 2/2 hosts (name filter:-0, limits:-0):
	  0: db01 (secrets: 0, health checks: 0)
	  1: web01 (secrets: 0, health checks: 0)

<probably lots of nix-build output>

/nix/store/grvny5ga2i6jdxjjbh2ipdz7h50swi1n-quetzal
nix result path:
/nix/store/grvny5ga2i6jdxjjbh2ipdz7h50swi1n-quetzal
```

The result path is written twice, which is a bit silly, but the reason is that only the result path is written to stdout, and everything else (including `nix-build` output) is redirected to stderr.
This makes it easy to use Quetzal for scripting, e.g. if one want to build using Quetzal and then `nix copy` the result path somewhere else.

Note that `examples/simple.nix` contain two different hosts definitions, and a lot of copy paste.
All the usual nix tricks can of course be used to avoid duplication.

Hosts can be deployed with the `deploy` command as follows:
`quetzal deploy examples/simple.nix` (this will fail without modifying `examples/simple.nix`).


### Selecting/filtering hosts to build and deploy

All hosts defined in a deployment file is returned to Quetzal as a list of hosts, which can be manipulated with the following flags:

- `--on glob` can be used to select hosts by name, with support for glob patterns
- `--limit n` puts an upper limit on the number of hosts
- `--skip n` ignore the first `n` hosts
- `--every n` selects every n'th host, useful for e.g. selecting all even (or odd) numbered hosts

(all relevant commands should already support these flags.)

The ordering currently can't be changed, but should be deterministic because of nix.

Most commands output a header like this:
```
Selected 4/17 hosts (name filter:-6, limits:-7):
	  0: foo-p02 (secrets: 0, health checks: 1)
	  1: foo-p05 (secrets: 0, health checks: 1)
	  2: foo-p08 (secrets: 0, health checks: 1)
	  3: foo-p11 (secrets: 0, health checks: 1)
```

The output is pretty self explanatory, except probably for the last bit of the first line.
`name filter` shows the change in number of hosts after glob matching on the hosts name, and `limits` shows the change after applying `--limit`, `--skip` and `--every`.


#### Tagging hosts

Each host can be tagged with an arbitrary amount of tags, which can be used to select and sort hosts.

To tag a host, use the `deployment.tags` option, e.g. `deployment.tags = [ "prod" "master" "rack-17" ]`. Hosts can now be selected with the `--tagged` option, e.g.`--tagged="prod,master"` will only select hosts tagged _both_ `prod` _and_ `master`.

To sort hosts based on tags, use the `network.ordering.tags` option, e.g. `network.ordering.tags = [ "master" "slave"]`. This ordering can be changed at runtime using the `--order-by-tags` option, eg. `--order-by-tags="slave,master"` (this also works when `network.ordering.tags` isn't defined). Hosts without matching tags will end up at the end of the list.


### Environment Variables

Quetzal supports the following (optional) environment variables:

- `SSH_IDENTITY_FILE` the (local) path to the SSH private key file that should be used
- `SSH_USER` specifies the user that should be used to connect to the remote system
- `SSH_SKIP_HOST_KEY_CHECK` if set disables host key verification
- `SSH_CONFIG_FILE` allows to change the location of the ~/.ssh/config file
- `QUETZAL_NIX_EVAL_CMD` Quetzal will invoke this command instead of default: "nix-instantiate" on PATH 
- `QUETZAL_NIX_BUILD_CMD` Quetzal will invoke this command instead of default: "nix-build" on PATH 
- `QUETZAL_NIX_SHELL_CMD` Quetzal will invoke this command instead of default: "nix-shell" on PATH
- `QUETZAL_NIX_EVAL_MACHINES` path to a custom eval-machines.nix. Defaults to the eval-machines.nix bundled with Quetzal

### Secrets

Files can be uploaded without ever ending up in the nix store, by specifying each file as a secret. This will use scp for copying a local file to the remote host.

See `examples/secrets.nix` or the type definitions in `data/options.nix`.

To upload secrets, use the `quetzal upload-secrets` subcommand, or pass `--upload-secrets` to `quetzal deploy`.

*Note:*
Quetzal will automatically create directories parent to `secret.Destination` if they don't exist.
New dirs will be owned by root:root and have mode 755 (drwxr-xr-x).
Automatic directory creation can be disabled by setting `secret.mkDirs = false`.


### Health checks

Quetzal has support for two types of health checks:

* command based health checks, which are run on the target host (success defined as exit code == 0)
* HTTP based health checks, which are run from the host Quetzal is running on (success defined as HTTP response codes in the 2xx range)

See `examples/healthchecks.nix` for an example.

There are no guarantees about the order health checks are run in, so if you need something complex you should write a script for it (e.g. using `pkgs.writeScript`).
Health checks will be repeated until success, and the interval can be configured with the `period` option (see `data/options.nix` for details).

It is currently possible to have expressions like `"test \"$(systemctl list-units --failed --no-legend --no-pager |wc -l)\" -eq 0"` (count number of failed systemd units, fail if non-zero) as the first argument in a cmd-healthcheck. This works, but is discouraged, and might break at any time.


### Pre-deploy checks (experimental)

Quetzal supports running checks before changing the target host (note: files will still be pushed to the host).
These checks work exactly like health checks, which means they will run forever until they have all succeeded.
This is an _experimental feature that is very likely to change_ in the future. Comments and feedback welcome :).

Pre-deploy checks can be defined using `deployment.preDeployChecks`.


### Advanced configuration

**nix.conf-options:** The "network"-attrset supports a sub-attrset named "nixConfig". Options configured here will pass `--option <name> <value>` to all nix commands.
Note: these options apply to an entire deployment and are *not* configurable on per-host basis.
The default is an empty set, meaning that the nix configuration is inherited from the build environment. See `man nix.conf`.

**network.buildShell**
By passing `--allow-build-shell` and setting `network.buildShell` to a nix-shell compatible derivation (eg. `pkgs.mkShell ...`), it's possible to make Quetzal execute builds from within the defined shell. This makes it possible to have arbitrary dependencies available during the build, say for use with nix build hooks. Be aware that the shell can potentially execute any command on the local system.

**special deployment options:**

(per-host granularity)

`buildOnly` makes Quetzal skip the "push" and "switch" steps for the given host, even if "quetzal deploy" or "quetzal push" is executed. (default: false)

`substituteOnDestination` Sets the `--substitute-on-destination` flag on nix copy, allowing for the deployment target to use substitutes. See `nix copy --help`. (default: false)


Example usage of `nixConfig` and deployment module options:
```
network = {
    nixConfig = {
        "extra-sandbox-paths" = "/foo/bar";
    };
};

machine1 = { ... }: {
    deployment.buildOnly = true;
};

machine2 = { ... }: {
    deployment.substituteOnDestination = true;
};
```

**mutually recursive configurations**
Each host's configuration has access to a `nodes` argument, which contains the compiled configurations of all hosts.

```
machine1 = { nodes, ... }: {
    hostnames.machine2 = 
        (builtins.head nodes.machine2.networking.interfaces.foo.ipv4.addresses).address;
    networking.interfaces.foo.ipv4.addresses = [
        {
            address = "10.0.0.10";
            prefixLength = 32;
        }
    ];
}

machine2 = { nodes, ... }: {
    hostnames.machine1 = 
        (builtins.head nodes.machine1.networking.interfaces.foo.ipv4.addresses).address;
    networking.interfaces.foo.ipv4.addresses = [
        {
            address = "10.0.0.20";
            prefixLength = 32;
        }
    ];
}
```


## Hacking Quetzal

All commands mentioned below is available in the nix-shell, if you run `nix-shell` with working dir = project root. The included `shell.nix` uses the latest `nixos-unstable` from GitHub by default, but you can override this by passing in another, eg. `nix-shell --arg nixpkgs '<nixpkgs>'` for your `$NIX_PATH` nixpkgs.

### Go dependency management

From within `nix-shell`, `go get -u` updates all go modules. Remember to update the `vendorSha256` in `./default.nix`

### Building the project with pinned dependencies

$ `nix-build --arg nixpkgs "builtins.fetchTarball https://github.com/NixOS/nixpkgs/archive/<rev>.tar.gz"`

## About the project

We needed a tool for managing our NixOS servers, and ended up writing one ourself. This is it. We use it on a daily basis to build and deploy our NixOS fleet, and when we need a feature we add it.

Quetzal is by no means done. The CLI UI might (and probably will) change once in a while.
The code is written by humans with an itch to scratch, and we're discussing a complete rewrite (so feel free to complain about the source code since we don't like it either).
It probably wont accidentally switch your local machine, so you should totally try it out, but do consider pinning to a specific git revision.
