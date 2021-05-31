#!/bin/bash
NFPM_EPOCH=$(git rev-list --all --count) $@
