#!/usr/bin/env bash

ZIF_BIN="$GOPATH/bin/zifd"
VERBOSE=""
NONPMINS=0
NOPAX=0

for arg; do
    case $arg in
        -h)
            echo "$(echo $0): install all Zif components."
            echo "Usage flags:"
            echo "	-h	Show this help text."
            echo "	-v	Be verbose."
            echo "	-n	Do not install the GUI part."
            echo "	-p	Do not change PaX flags of the executable (when PaX is detected)."
            exit
            ;;
        -v)
            VERBOSE="-v -x"
            ;;
        -p)
            NOPAX=1
            ;;
        -*)
            echo "invalid option: $arg"
            exit
            ;;
        *)
            echo "what is '$arg' supposed to be?"
            exit
            ;;
    esac
done


pushd libzif
go install $VERBOSE

pushd data
go install $VERBOSE

popd
popd

pushd zifd
go install $VERBOSE
popd

if [ $NOPAX -eq 0 ]; then
    if [ -d "/proc/sys/kernel/pax" ]; then
        paxctl -c "$ZIF_BIN" && setfattr -n user.pax.flags -v "emr" "$ZIF_BIN"
    fi
fi

