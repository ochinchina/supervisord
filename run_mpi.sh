#!/bin/bash
export PMIX_SERVER_URI=$(cat dvm_uri)

prte --report-uri dvm_uri
