{
  description = "DevShell for the WWW project";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
  };
  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });

      commandPrefix = "sb-";
      commands = 
      let
        httpServerGardenCenter = "python3 -m http.server 8065 -b localhost --directory ./www/sramek-garden-center/";
        httpServerTransportation = "python3 -m http.server 8066 -b localhost --directory ./www/sramek-transportation/";
      in [
        # Code
        {
          name = "build-go";
          command = "go build";
        }
        {
          name = "fmt-go";
          command = "go fmt";
        }
        {
          name = "fmt-html";
          # ignore errors as tidy returns 1 on warnings
          command = "sh -c \''for f in $(find . -name \"*.html\"); do tidy -m -i -c $f || true; done\''";
        }
        {
          name = "fmt-all";
          command = "fmt-go && fmt-html";
        }
        # Infra
        {
          name = "deploy";
          command = "pulumi stack select prod && pulumi up";
        }
        {
          name = "start-http-server";
          command = "${httpServerGardenCenter} && ${httpServerTransportation}";
        }
      ];
    in 
    {
      devShells = forEachSupportedSystem ({pkgs}: {
        default = pkgs.mkShell {
          packages = with pkgs; [
              awscli2
              go
              html-tidy
              nodejs_22
              python3
              pulumi
              pulumiPackages.pulumi-language-go
              python3
              html-tidy
          ];

          shellHook = with pkgs.lib; let 
            setAlias = command: "alias ${commandPrefix}${command.name}='${command.command}'";
            setAliases = forEach commands (command: setAlias command);
          in ''
          echo "Welcome to the devshell!"

          # Set aliases
          ${builtins.concatStringsSep "\n" setAliases} 
          '';
        };
      });
    };
}
