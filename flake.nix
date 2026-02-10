{
  description = "age-plugin-agent - Age plugin proxy agent for remote encryption/decryption over SSH";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      # Package derivation as a function of pkgs, allowing overlay consumers
      # to build against their own nixpkgs instance
      mkPackage = pkgs: pkgs.buildGoModule {
        pname = "age-plugin-agent";
        version = "0.1.0";
        src = self;
        # No external Go module dependencies
        vendorHash = null;

        meta = with pkgs.lib; {
          description = "Age plugin proxy agent that forwards encryption/decryption to a remote server";
          homepage = "https://github.com/age-plugin-agent";
          license = licenses.mit;
          platforms = platforms.unix;
          mainProgram = "age-plugin-agent";
        };
      };

      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
    in
    flake-utils.lib.eachSystem supportedSystems (system:
      let
        pkgs = import nixpkgs { inherit system; };
        package = mkPackage pkgs;
      in
      {
        packages = {
          age-plugin-agent = package;
          default = package;
        };

        apps = {
          age-plugin-agent = {
            type = "app";
            program = "${package}/bin/age-plugin-agent";
          };
          default = self.apps.${system}.age-plugin-agent;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gotools
            gopls
          ];
        };
      }
    ) // {
      # Overlay for inclusion in other flakes:
      #
      #   inputs.age-plugin-agent.url = "github:...";
      #
      #   nixpkgs.overlays = [ inputs.age-plugin-agent.overlays.default ];
      #
      # Then use pkgs.age-plugin-agent anywhere in your config.
      overlays.default = final: _prev: {
        age-plugin-agent = mkPackage final;
      };
    };
}
