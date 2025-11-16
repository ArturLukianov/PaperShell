all: clean
	@ echo "    * Building agent_papershell plugin"
	@ mkdir dist
	@ cp config.json ax_config.axs ./dist/
	@ cp -r src_papershell ./dist/
	@ go build -buildmode=plugin -ldflags="-s -w" -o ./dist/agent_papershell.so pl_main.go pl_agent.go
	@ echo "      done..."

clean:
	@ rm -rf dist
