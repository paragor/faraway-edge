build_local:
	goreleaser release --snapshot --clean

update_default_config:
	rm config_example.yaml || true
	go run main.go example  > config_example.json
