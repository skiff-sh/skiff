#!/bin/bash

# go_local_update_final.sh - Gets the latest git hash from a local clone of a module
#                            and forces the current project to use that specific hash.
#
# Usage: ./go_local_update_final.sh [path/to/local/library] [module/import/path]
# Example: ./go_local_update_final.sh ../my-sdk github.com/myorg/my-sdk

# --- Configuration ---
LIBRARY_LOCAL_PATH_ARG="$1"   # First argument: relative/absolute path to the local clone
LIBRARY_MODULE_PATH="$2"      # Second argument: the Go module import path (e.g., github.com/user/repo)
GO_MOD_FILE="go.mod"

# --- Functions ---

# Function to resolve the path and check environment
check_and_resolve_path() {
    if [ -z "$LIBRARY_LOCAL_PATH_ARG" ] || [ -z "$LIBRARY_MODULE_PATH" ]; then
        echo "Error: Missing arguments."
        echo "Usage: $0 <path/to/local/library> <module/import/path>"
        exit 1
    fi

    # 1. Resolve the user-provided path to an absolute path for reliability

    # Use standard shell commands (cd and pwd) for maximum portability
    LIBRARY_LOCAL_PATH=$(cd "$LIBRARY_LOCAL_PATH_ARG" && pwd)

    if [ -z "$LIBRARY_LOCAL_PATH" ]; then
        echo "Error: Could not resolve path to a valid directory: $LIBRARY_LOCAL_PATH_ARG"
        exit 1
    fi

    # 2. Validation checks using the absolute path
    if [ ! -d "$LIBRARY_LOCAL_PATH/.git" ]; then
        echo "Error: Directory is not a Git repository: $LIBRARY_LOCAL_PATH"
        exit 1
    fi
    if [ ! -f "$GO_MOD_FILE" ]; then
        echo "Error: $GO_MOD_FILE not found. Are you in the root directory of your project?"
        exit 1
    fi
}

# --- Main Execution ---
check_and_resolve_path

echo "--- Starting Local Hash Update ---"
echo "Local Path: $LIBRARY_LOCAL_PATH"
echo "Module Path: $LIBRARY_MODULE_PATH"

# 1. Get the latest commit hash from the local library repo
echo "1. Determining latest commit hash..."

# Store the current directory to ensure we can return safely
CURRENT_DIR=$(pwd)

# Change directory to the library clone to run git
cd "$LIBRARY_LOCAL_PATH" || exit 1 # Exit if cd fails

# Get the full 40-character commit hash
LATEST_HASH=$(git rev-parse HEAD)

# Change back to the original project directory
cd "$CURRENT_DIR" || exit 1

if [ -z "$LATEST_HASH" ]; then
    echo "Error: Failed to retrieve commit hash."
    exit 1
fi

echo "   -> Latest Commit Hash Found: $LATEST_HASH"

# 2. Run go get with the specific hash
echo "2. Running go get: $LIBRARY_MODULE_PATH@$LATEST_HASH"
go get "$LIBRARY_MODULE_PATH"@"$LATEST_HASH"

if [ $? -ne 0 ]; then
    echo "Error: 'go get' failed. Check your network or module path."
    exit 1
fi

echo "--- Update Complete. Project is now pinned to commit $LATEST_HASH. ---"
