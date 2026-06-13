{ pkgs, lib, config, ... }:

{
  # https://devenv.sh/basics/
  env.SOULSOLID_CONFIG_PATH = "./config.yaml";

  # https://devenv.sh/packages/
  packages = [
    pkgs.tailwindcss
    pkgs.tree
    pkgs.chromaprint # fpcalc, used for audio fingerprinting
    pkgs.flac
    pkgs.id3v2
  ];

  # https://devenv.sh/languages/
  languages.go.enable = true;

  languages.javascript = {
    enable = true;
    package = pkgs.nodejs_24;
    npm = {
      enable = true;
      install.enable = true; # runs `npm install` (= `npm ci` when a lockfile exists)
    };
  };

  # https://devenv.sh/scripts/
  scripts.build-assets.exec = ''
    npm run build:css
    npm run build:assets
    npm run copy:deps
  '';

  # https://devenv.sh/processes/
  # Run with: devenv up
  processes.soulsolid.exec = "npm run dev";

  # https://devenv.sh/basics/
  enterShell = ''
    build-assets
  '';
}
