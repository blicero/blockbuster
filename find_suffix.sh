#!/bin/sh
# Time-stamp: <2021-08-05 20:36:18 krylon>
#
# Find the unique filename suffixes in a list of directory trees.
# smh - I should either write this in pure Perl or as a pure shell script.
# But it does what it's supposed to, so ...

find $@ -type f  | perl -e 'while(<>) { chomp; if (/([.]([^.]+)$)/) { print "$1\n"; } } ' | sort | uniq
