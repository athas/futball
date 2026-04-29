with import <nixpkgs> {};

mkShell {
  buildInputs = [
    ocl-icd
    opencl-headers
    python313
    python313Packages.pygame
    python313Packages.numpy
    python313Packages.pyopencl
    SDL2
    SDL2_ttf
    futhark
  ];
}
