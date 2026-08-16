# City Lighting Center

这是一个纯 Go 城市照明遥测中心。路灯节点上报节点编号、区域编号、电压、亮度和故障码，并用中心密钥生成 HMAC 签名。区域编号会作为独立的附加认证数据参与签名，因此改动区域后消息无法通过验证。中心使用内存仓储保存通过校验的消息，并返回批次总数、有效数和零基失败行号。

## 环境

项目使用 Go 1.23.12，`go.mod` 的语言版本是 1.23。执行命令前设置：

```sh
export GOTOOLCHAIN=local
```

项目兼容 `CGO_ENABLED=0`，不需要数据库、外部服务或随机源。

## 运行

直接运行会处理内置的三条固定夹具并输出批次统计：

```sh
GOTOOLCHAIN=local CGO_ENABLED=0 go run ./cmd/lighting-center
```

也可以使用 `-stdin` 从标准输入传入 JSON 数组：

```sh
printf '%s\n' '[{"node_id":"lamp-1","region_id":"region-a","voltage":"220","brightness":"80","fault_code":"OK","signature":"..."}]' | GOTOOLCHAIN=local go run ./cmd/lighting-center -stdin
```

失败行号从 0 开始，因此三条消息的最后一条行号为 2。

## 测试

```sh
GOTOOLCHAIN=local CGO_ENABLED=0 go test -count=1 ./...
```

## 构建

```sh
GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...
GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./...
```

`internal/lighting` 包含签名、批量验证、失败统计和内存仓储，`cmd/lighting-center` 是可执行入口。
