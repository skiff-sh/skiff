#!/bin/bash

# Define the directory to search within
TARGET_DIR="pkg/plugin/testdata"

# Loop through all items matching the extension pattern within the directory
for file_path in "$TARGET_DIR"/*.go; do

    # 1. CRUCIAL CHECK: Ensure the path points to an actual file.
    if [[ -f "$file_path" ]]; then

        echo "Processing file: $file_path"

        # --- PURE BASH PATH MANIPULATION ---

        # 2a. Get the filename with extension (e.g., 'my_plugin.go')
        #    Use ##*/ to remove the longest match of the path (everything up to the last /)
        file_name_with_ext="${file_path##*/}"

        # 2b. Get the filename without extension (e.g., 'my_plugin')
        #    Use %.* to remove the shortest match of the extension from the end
        file_name_no_ext="${file_name_with_ext%.*}"

        # 3. Define the output WASM file path
        # Output: pkg/plugin/testdata/my_plugin.wasm
        WASM_OUTPUT_PATH="$TARGET_DIR/$file_name_no_ext.wasm"

        # 4. Run the Go build command
        GOOS=wasip1 GOARCH=wasm go build \
            -buildmode=c-shared \
            -o "$WASM_OUTPUT_PATH" \
            "$file_path"

        echo "   -> Built to: $WASM_OUTPUT_PATH"
    fi
done

echo "Build process complete."

echo "Zipping files for archive tests"

tar -czvf pkg/artifact/testdata/source.tar.gz -C pkg/artifact/testdata source
cd pkg/artifact/testdata && rm -f source.zip && zip -ry source.zip source
