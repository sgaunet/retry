#!/usr/bin/env bash
random=$(( ( RANDOM % 3 )  + 1 ))
echo "Random number: $random"
if [ $random -eq 1 ]; then
  exit 0
fi
exit 1
