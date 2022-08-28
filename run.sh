#! /bin/sh
# rebuild apps if necessary
make compile
# run app with some arguments
./bin/main-linux-386 "$@"