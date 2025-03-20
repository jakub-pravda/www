#!/usr/bin/env bash

# Pre commit checks to ensure code quality and formatting

function error_with_message {
    echo $1
    exit 1
}

function run_html_tidy {
    echo "Running HTML Tidy on $1"
    find ./www/$1/ -name "*.html" | while read -r f; do
        echo "$f"
        tidy --custom-tags blocklevel -m -i -c "$f" || true
    done
}

CODE_ROOT_DIR="$(pwd)/.."

echo "Running pre-commit checks"

## Check golang source code
echo "Checking golang source code"
cd $CODE_ROOT_DIR || error_with_message "Failed to change directory to $CODE_ROOT_DIR"

if ! command -v go &> /dev/null
then
    error_with_message "go could not be found, please install go"
fi

cd ./infra/src || error_with_message "Failed to change directory to infra/src"
go fmt
go build -o /dev/null

## Check html
echo "Checking HTML files"
cd $CODE_ROOT_DIR || error_with_message "Failed to change directory to $CODE_ROOT_DIR"

if ! command -v tidy &> /dev/null
then
    error_with_message "tidy could not be found, please install tidy"
fi

run_html_tidy "sramek-garden-center"