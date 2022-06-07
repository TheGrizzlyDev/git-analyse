# Git Analyse

This tool allows you to run actions in parallel on a group of commits. 

## Current features

- implements a functional clone of 'git bisect'

## TODO

Pretty much everything else, I only worked on this a few hours while trying to sleep. But basically I still need to:
- implement a 'benchmark' subcommand that runs a benchmark for each and every commit and returns some kind of stats
- implement a 'clean' action
- make this work against Bazel's RBE, which would make it even more parallelizable and furthermore would require little to no upload as most inputs will probably already be available on the CAS
- try to predict how much time is left for both benchmark and bisect
- start caching outputs