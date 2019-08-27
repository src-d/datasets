#!/bin/bash
if [ -f "$1/etc/os-release" ]; then
ID=$(grep -oP '(?<=^ID=).+' $1/etc/os-release | tr -d '"')
VERSION=$(grep -oP '(?<=^VERSION_ID=).+' $1/etc/os-release)
VERSION=$(sed -e 's/^"//' -e 's/"$//' <<<"$VERSION")
else
if ls $1/etc/*release 1> /dev/null 2>&1; then 
ID=$(cat $1/etc/*release  | tr '[:upper:]' '[:lower:]' | cut -f 1 -d ' ' | head -n 1)
fi
fi

echo $ID:$VERSION