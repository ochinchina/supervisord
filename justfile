dirpath := justfile_directory()

sync target="frp":
    rsync --delete -avP "{{ dirpath }}/" {{ target }}:$(basename "{{ dirpath }}")/ --exclude-from=.rsync_exclude

build:
  mkdir -p build
  rm -rf build/*  
  go build -tags release -a -ldflags "-linkmode external -extldflags -static" -o build/supervisord .
