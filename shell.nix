with import <nixpkgs> {};

mkShell {
  buildInputs = [
    ocl-icd
    opencl-headers
    python3
    python3Packages.pygame
    python3Packages.numpy
    python3Packages.pyopencl
    SDL2
    SDL2_ttf
  ];
}
