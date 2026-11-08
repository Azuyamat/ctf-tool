{
  description = "Manage CTF challenge folders from the terminal";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      version = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        rec {
          # go.mod requires go >= 1.27.1, newer than the default `go` in
          # this nixpkgs snapshot; buildGo127Module pins a matching toolchain.
          ctf = pkgs.buildGo127Module {
            pname = "ctf";
            inherit version;

            src = ./.;
            vendorHash = "sha256-uT9amskE6iyJonki225evxJdxLFren0i9JXGVjNHZD0=";

            subPackages = [ "cmd/ctf" ];

            env.CGO_ENABLED = 0;
            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];

            meta = {
              description = "Manage CTF challenge folders from the terminal";
              homepage = "https://github.com/Azuyamat/ctf-tool";
              mainProgram = "ctf";
            };
          };
          default = ctf;
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_27
              pkgs.gopls
              pkgs.golangci-lint
            ];
          };
        }
      );
    };
}
