# gopm

This is a hard fork of [supervisord](https://github.com/ochinchina/supervisord) with the intention
of using it strictly to manage a group of processes for local microservice development.

# Installation

```
$ make install
```

# Building in Development

```
$ make
```

The resulting binaries will be in `bin/` directory. These binaries will seek out
the `./webgui` assets on disk when rendering the web UI.

If you'd like to build for release, set `RELEASE` environment variable like so:

```
$ make RELEASE=1
```

This will include the `release` build tag and embed any assets in the binaries.
