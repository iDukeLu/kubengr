# 更新项目依赖
tidy: script-check
	go mod tidy

# 检查 shell 脚本
script-check:
	shellcheck -s bash -f json  ./scripts/kubengr.sh | jq

# 通过脚本生成模版
gen:
	python3 -c 'template = open("./scripts/script_template.txt").read(); content = open("./scripts/kubengr.sh").read(); result = template.replace("{{CONTENT}}", content); print(result)' > ./scripts/script.go

# 构建 linux 二进制
build-linux: gen
	CGO_ENABLED=0  GOOS=linux  GOARCH=amd64  go build cmd/app/kubengr.go

# 上传二进制文件
upload: build-linux
	scp kubengr root@$$ip:~/ 

