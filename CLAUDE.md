# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

ApiBatchRequestor (abr) — 单机版 API 批量请求/压测命令行工具。从源文件(每行一条数据)按配置构造 HTTP GET/POST 请求，支持限速、限并发、QPS 控制、结果解析存储、错误记录。零外部环境依赖(单二进制可运行)。

模块名: `apiBatchRequester`,Go 1.18。

## 常用命令

```bash
# 跑单个包测试
go test ./core/...

# 跑单个测试函数
go test -run TestTemplateReplace ./core/

# 跨平台构建(产物即根目录下的 abrForLinux/abrForMac_amd/abrForMac_arm)
./build.sh

# 单平台构建
CGO_ENABLED=0 go build -o apiBatchRequester

# 运行
./apiBatchRequester -f ./zDemoConfs/postDemo.yaml

# 启动本地测试服务(8081 端口,提供 /getDemo 和 /postDemo)
cd testServer && go run demoServer.go
```

## 架构总览

代码量小，采用单文件核心 + 钩子注入模式。

```
main.go                  # 入口:解析配置(json/yaml)、配置全局 DefaultHttpClient、启动 poster.Run()
core/
  abr.go                 # apiPoster 主体, 生产消费模型(行 216-285 producer / 行 618-703 Run 调度)
  abrConfStruct.go       # ApiPosterConf 全部配置项
  header.go              # 动态请求头解析(支持 $n 占位符)
  httpClient.go          # 全局 DefaultHttpClient(单例)
  utils.go               # appendParamsToURL (GET 参数合并)
hooks/
  paramBuilder/common.go    # 注入点: 自定义参数构造函数
  paramAppender/common.go   # 注入点: 自定义参数补充函数(GET 拼 query / POST merge body)
testServer/demoServer.go    # 本地测试服务(8081)
zDemoConfs/                  # 配置样例 + 测试数据
```

### 核心数据流(生产-消费)

`apiPoster.Run()` 启动三类 goroutine 通过 channel 协作:

1. **itemProducerRun** (单协程) — 读取 `SrcFilePath` 逐行(或按 `\x1E` 行分隔符)放入 `c.ch`
2. **Worker 协程** (`WorkerCoroutineNum` 个) — 从 `c.ch` 消费,调用 `itemGet`/`itemPost` 发起请求
3. **errItemSaverRun / resItemSaverRun** (单协程) — 从 `c.errItemCh`/`c.resCh` 消费并落盘

QPS 控制由 `QpsLimit` + `QPerTimeRange` 在 producer 内 `time.Sleep` 实现;并发度由 `WorkerCoroutineNum` 控制(超过 `MaxCoroutineNum=3000` 会被截断)。

### 配置加载

`ApiPosterConf` 是单层平铺结构,所有字段同时支持 `json` 和 `yaml` tag。运行时配置转换/校验集中在 `NewPoster()` ([core/abr.go:59](core/abr.go#L59)) — 这里设置默认值、解析 `Header` 字符串、`BuiltInParamBuilder`/`BuiltInParamAppender` 查表绑定等。

### 请求参数构造(可叠加)

`core/abr.go` 中 `itemPost`/`itemGet` 按以下优先级选用构造方式:

1. `ParamDirect=true` — 文件每行直接当作参数
2. `PostParamTemplate` / `GetParamTemplate` — 旧版 `%s` 模板
3. `PostParamTemplateV2` / `GetParamTemplateV2` — 新版 `$n` / `$0` / `$.JSONn` 模板
4. `BuiltInParamBuilder` — 钩子函数(运行时按 `BuiltInParamBuilderName` 查表)
5. `BuiltInParamAppender` — 钩子函数, 在上述方式之后补充(GET 拼 query / POST merge 到 body 顶层)

模板语法在 `TemplateReplace` ([core/abr.go:705](core/abr.go#L705)),支持 `$0` 整行 / `$n` 第 n 列 / `$.JSONn` JSON 编码。

## 扩展指南

### 添加新的参数构造钩子

编辑 [hooks/paramBuilder/common.go](hooks/paramBuilder/common.go) 的 `init()`,往 `BuiltInParamBuilderNameMap` 注册新 key。示例已有 `demo`。配置文件中通过 `builtInParamBuilderName: <key>` 启用。

### 添加新的参数补充钩子

编辑 [hooks/paramAppender/common.go](hooks/paramAppender/common.go) 的 `init()`,往 `BuiltInParamAppenderNameMap` 注册新 key。示例已有 `time`(自动加时间戳)。

注意: 钩子改动后必须重新编译二进制(见 `build.sh`)。

### 添加新配置项

1. 在 [core/abrConfStruct.go](core/abrConfStruct.go) 的 `ApiPosterConf` 加字段(同时给 json/yaml tag)
2. 如需默认值或校验,在 `NewPoster()` 里处理
3. 在 [readme.md](readme.md) 的配置项清单追加说明

## 编码约定(来自全局 CLAUDE.md)

- **不要用魔法数字**: 阈值/分隔符等字面量必须抽成包内 const(已用示例: `MaxCoroutineNum`、`LineSeparatorDefault`)
- **全局单例**: 状态无关就用包函数(如 `TemplateReplace`),有状态且无依赖就在 `init()` 初始化(已用示例: `DefaultHttpClient`、`BuiltInParamBuilderNameMap`)
- **错误处理**: `main.go` 在启动期 `log.Fatal`;运行期错误用 `log.Println` + 写 `errItemCh`,不中断流程
- **本项目无 webapi**: 不需要 `ginHandler + service + dao` 架构

## 易踩坑

- `Header` 字段是单个字符串,用 `\t` 分隔多个头,值支持 `$n` 引用文件行第 n 列(由 `ParseReqHeader` 解析, `CompleteReqHeader` 在请求前填充)
- `MaxCoroutineNum=3000` 是硬上限,调高需改 `core/abr.go` 常量
- `timeLimit`(秒)到点会停,但已完成请求会正常落盘;中断恢复目前只能靠 `srcFileSkip` 跳过已处理行
- `dryRun=true` 时不实际发请求,只是打印构造结果(配置里如果设了 dryRun,代码会强制把 `DetailLog` 也打开)
- `ReqHost` 字段用于覆盖 HTTP 请求的 Host 头(用于多租户/虚拟主机场景),不是改实际 DNS
