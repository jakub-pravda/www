{
  description = "DevShell for the WWW project";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };
  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });
    in 
    {
      devShells = forEachSupportedSystem ({pkgs}: {
        default = pkgs.mkShell {
          packages = with pkgs; [
              awscli2
              go
              go-task
              html-tidy
              nodejs_22
              python3
              pulumi
              pulumiPackages.pulumi-language-go
              python3
              html-tidy
          ];
        };
      });
    };
}
