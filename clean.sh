#!/bin/sh
# Time-stamp: <2021-08-06 15:20:44 krylon>

cd $GOPATH/src/github.com/blicero/blockbuster/

rm -vf bak.blockbuster blockbuster dbg.build.log && \
    du -sh . && \
    git fsck --full && \
    git reflog expire --expire=now && \
    git gc --aggressive --prune=now && \
    du -sh .

