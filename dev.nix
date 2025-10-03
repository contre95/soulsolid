{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.nodejs_24
    pkgs.tailwindcss
    pkgs.tree
  ];
  
  shellHook = ''
    echo "Setting up Soulsolid development environment"
    npm install
    npm run build:css
    npm run build:assets
    npm run copy:deps
  '';
}

