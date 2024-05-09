clean:
	@find ${WORKING_DIR} -depth -type d \
		\( -name 'bin' -o -name 'obj' \) \
		-exec rm -rf '{}' + \
		-ls

.PHONY: brokerc
brokerc:
	@echo 'Experimental SDK container build: https://github.com/dotnet/sdk-container-builds'
	cd Broker && dotnet publish \
		--os linux \
		--arch x64 \
		--configuration Release \
		/p:PublishProfile=DefaultContainer
