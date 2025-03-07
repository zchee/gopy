# syntax=docker/dockerfile-upstream:master-labs
# check=error=true

ARG GOLANG_VERSION=1.24

FROM golang:${GOLANG_VERSION}-bookworm AS builder
ARG TARGETOS
ARG TARGETARCH

ARG PYTHON_VERSION=3.13.2

WORKDIR /

# Install dependencies for build Python freethreading
# RUN rm -f /etc/apt/apt.conf.d/docker-clean; echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/docker-keep-cache
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked --mount=type=cache,target=/var/lib/apt,sharing=locked \
    set -eux; \
    apt-get update; \
    apt-get install -y \
        build-essential \
        libbz2-dev \
        libffi-dev \
        libgdbm-compat-dev \
        libgdbm-dev \
        liblzma-dev \
        libncurses5-dev \
        libreadline-dev \
        libreadline6-dev \
        libsqlite3-dev \
        libssl-dev \
        lzma \
        lzma-dev \
        pkg-config \
        tk-dev \
        uuid-dev \
        wget \
        zlib1g-dev

# Install LLVM 19 to build with --enable-experimental-jit
RUN wget -qO- https://apt.llvm.org/llvm-snapshot.gpg.key | tee /etc/apt/trusted.gpg.d/apt.llvm.org.asc; \
  echo "deb [signed-by=/etc/apt/trusted.gpg.d/apt.llvm.org.asc] http://apt.llvm.org/bookworm/ llvm-toolchain-bookworm-19 main" | tee /etc/apt/sources.list.d/llvm.list

RUN set -eux; \
    apt-get update; \
    apt-get install -y \
        # clang
        clang-19 libclang-common-19-dev libclang-19-dev libclang1-19 python3-clang-19 \
        # compiler-rt
        libclang-rt-19-dev \
        # lld
        lld-19 \
        # libc++
        libc++-19-dev libc++abi-19-dev \
        # OpenMP
        libomp-19-dev \
        # LLVM
        libllvm19 llvm-19 llvm-19-dev llvm-19-runtime \
        # mlir
        libmlir-19-dev mlir-19-tools \
        # wasm support
        libclang-rt-19-dev-wasm32 libclang-rt-19-dev-wasm64 libc++-19-dev-wasm32 libc++abi-19-dev-wasm32 libclang-rt-19-dev-wasm32 libclang-rt-19-dev-wasm64 \
        # LLVM libc
        libllvmlibc-19-dev \
        ; \
    rm -rf /var/lib/apt/lists/*

ENV PATH=/usr/lib/llvm-19/bin:$PATH

RUN wget -qO- https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz | tar -xzf -
RUN mkdir -p Python-${PYTHON_VERSION}/release
RUN sed -i -e 's|_LLVM_VERSION = 18|_LLVM_VERSION = 19|' Python-${PYTHON_VERSION}/Tools/jit/_llvm.py
RUN cd Python-${PYTHON_VERSION}/release; \
    ../configure \
		--prefix=/usr \
		--enable-ipv6 \
		--enable-loadable-sqlite-extensions \
		--enable-optimizations \
		--with-lto \
		--with-system-expat \
		--with-system-libmpdec \
		--with-readline=readline \
		--with-dbmliborder=ndbm \
		py_cv_module__tkinter=disabled \
		\
		--disable-gil \
		--enable-experimental-jit=yes \
		--with-mimalloc=yes \
		\
		CC='clang-19' \
		CXX='clang++-19 -stdlib=libc++' \
		CPPFLAGS='-I/usr/lib/llvm-19/include -I/usr/lib/llvm-19/include' \
		CFLAGS_NODIST='-I/usr/lib/llvm-19/include -I/usr/lib/llvm-19/include/c++' \
		LDFLAGS='-L/usr/lib/llvm-19/lib -L/usr/lib/llvm-19/lib/clang/19/lib/linux -rtlib=compiler-rt -fuse-ld=lld' \
		LDFLAGS_NODIST='-L/usr/lib/llvm-19/lib -L/usr/lib/llvm-19/lib/clang/19/lib/linux -Wl,-rpath,/usr/lib/llvm-19/lib -rtlib=compiler-rt -fuse-ld=lld'

RUN make -C Python-${PYTHON_VERSION}/release -j$(nproc)
RUN make -C Python-${PYTHON_VERSION}/release -j$(nproc) altinstall

# RUN make -j$(nproc) altinstall; \
# 	  cp ./libpython3.13d.so /usr/lib/x86_64-linux-gnu/libpython3.13d.so; \
# 	  cp ./libpython3.13d.so.1.0 /usr/lib/x86_64-linux-gnu/libpython3.13d.so.1.0
#
# RUN --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
# 			go install -a -v -gcflags "-N -l" std; \
# 			go install -v golang.org/x/tools/cmd/goimports@latest
#
# RUN --mount=type=cache,target=/root/.cache/pip,sharing=locked \
# 			python3.13 -m pip install -v pybindgen
#
# ENV LD_LIBRARY_PATH=Python-${PYTHON_VERSION}/release

# CMD /go/bin/gopy
