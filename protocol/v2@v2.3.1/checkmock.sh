#!/bin/bash

make mockgen
changes=$(git status -s)
if [ ! -n "$changes" ]; then
    echo "all mock files are newest"
else
    echo "mock file need update" >& 2
    echo $changes
    exit 1
fi