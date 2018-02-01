#!/bin/sh

# using https://github.com/mixu/markdown-styles 
# library to generate html output

generate-md --layout github --input ./md/index.md --output ./web
cp -r ./md/assets/* ./web/assets
