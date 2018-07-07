all: engine.py

run: engine.py
	python futball.py

engine.py: engine/*.fut
	futhark-pyopencl --library engine/engine.fut -o engine

engine.c: engine/*.fut
	futhark-opencl --library engine/engine.fut -o engine

_engine.so: engine.c
	build_futhark_ffi engine

clean:
	rm -f *.c *.o *.so *.pyc engine.py
