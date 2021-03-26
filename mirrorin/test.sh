#!/usr/bin/env bash
#set -e
set -u
set -o pipefail

url="${1}"
server="${url%\/*}"
path="${url#*\/}"
pattern='(?s)301 Moved Permanently.*\n.*Location: https:\/\/cookpad.com\/'
pattern="${pattern}${path}"

curl -v "${server}"/"${path}" 2>&1 | grep -oq -e "${pattern}"


if [ $? -eq 0 ]; then
  echo "Request to ${1} is redirected"
else
  echo "Request to ${1} was not forwarded to cookpad.com"
fi

