# using https://github.com/mixu/markdown-styles 
# library to generate html output
web/index.html: md/index.md web/assets
	generate-md --layout github --input ./md/index.md --output ./web
web/assets: md/assets
	mkdir -p ./web/assets
	cp -r ./md/assets/* ./web/assets/
web/clean:
	rm -rf ./web
