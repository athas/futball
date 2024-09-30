with import <nixpkgs> {};

mkShell {
  buildInputs = [
    ocl-icd
    opencl-headers
    python311
    python311Packages.pygame
    python311Packages.numpy
    python311Packages.pyopencl
    SDL2
    SDL2_ttf
    futhark
  ];
}
