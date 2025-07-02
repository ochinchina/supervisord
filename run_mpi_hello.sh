#!/bin/bash
export PMIX_SERVER_URI=$(cat dvm_uri)

sleep 1
prun --dvm-uri file:dvm_uri --np 4 python hello.py
