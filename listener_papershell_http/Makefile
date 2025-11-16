all: clean
	@ echo "    * Building listener_papershell_http plugin"
	@ mkdir dist
	@ cp config.json ax_config.axs ./dist/
	@ go build -buildmode=plugin -ldflags="-s -w" -o ./dist/papershell_http.so pl_main.go pl_listener.go pl_http.go
	@ echo "      done..."

clean:
	@ rm -rf dist
