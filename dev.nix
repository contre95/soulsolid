{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.nodejs_24
    pkgs.tailwindcss
    pkgs.go
    pkgs.tree
    pkgs.flac
    pkgs.id3v2
  ];
  
  shellHook = ''
    echo "Setting up Soulsolid development environment"
    npm install
    npm run build:css
    npm run build:assets
    npm run copy:deps
  '';
}

