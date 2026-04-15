{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };
  outputs = {
    self,
    nixpkgs,
  }: let
    system = "x86_64-linux";
    pkgs = import nixpkgs {inherit system;};
  in {
    devShells.${system}.default = pkgs.mkShell {
      buildInputs = with pkgs; [
        go_1_26
        golangci-lint
        gnumake
        zig
      ];
      shellHook = ''
        export GOPATH="$HOME/.local/share/go"
        export GOBIN="$GOPATH/bin"
        export PATH="$GOBIN:$PATH"
        export PATH="$HOME/.local/bin:$PATH"
        export ZIG_GLOBAL_CACHE_DIR="/tmp"
      '';
    };
  };
}
