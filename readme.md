# ApiBatchRequestor（abr）  单机版api批量请求跑批工具

```
./apiBatchRequester  -f  ./demo.yaml   
# abr,启动!!!
# 后台运行：nohup ./apiBatchRequester  -f  ./demo.yaml   >  demo.log 2>&1 &
# 配置文件支持json和yaml格式, apiBatchRequester无环境依赖，可直接下载可执行文件使用
```

## 简介

    		abr是一个用于批量构造请求参数发起http GET、POST（当前支持json参数）请求的命令行工具，包含数据源文件解析、参数构造、 批量请求、限速限并发、耗时统计、错误检查与记录、结果解析和存储等功能点。
    		abr核心逻辑是根据配置解析数据文件并逐行构造请求参数发起http请求，在4核低压cpu的Mac上能发起超1万qps的请求。  
    		你可以用abr来实现数据跑批、接口清洗、压测等任务。

## 最简实践
### 示例1：根据一个csv表格构造post请求

目标post接口与返回值示例：
```shell
POST  https://abr.test.com/userInfo/setInfo
参数实例
{"id":123,"name":"张三"}
返回结果
{
  "errNo":0,
  "msg":"succ"
  "data":{}
}

```
数据源文件： idName.csv
```
id,name 
123,张三
124,张四
125,张五
```
配置文件  getUserInfo.yaml
```yaml
# 基本参数
method: POST
host: https://abr.test.com
path: /userInfo/setInfo
# 参数构造语法
postParamTemplateV2: '{"id":"$1", "name":$2, "otherParam":"abc"}'  #详细构造语法见后文，这里$1表示文件某行第一列
# 源文件
srcFilePath: ./idName.csv     #数据源文件
# 并发控制
workerCoroutineNum: 50       #并发数，最多有多少个并发同时请求
qpsLimit: 100                #限速，每秒最多多少个请求
# 错误检查
resErrNoName: errNo          #错误探针，除了http错误外，还可以检查接口结果中的字段，非0触发错误
# 需要记录请求结果话见后文配置 
```

运行abr将构造并发送3个httpPost请求
```
POST https://abr.test.com/userInfo/setInfo  {"id":"123", "name":"张三", "otherParam":"abc"}
POST https://abr.test.com/userInfo/setInfo  {"id":"124", "name":"张四", "otherParam":"abc"}
POST https://abr.test.com/userInfo/setInfo  {"id":"125", "name":"张五", "otherParam":"abc"}
```

### 示例2：根据一个id列表文件get请求并保存结果数据

目标get接口与返回值实例：
```shell
GET  https://abr.test.com/userInfo/getById?id=123

返回结果
{
  "errNo":0,
  "data":{
    "id":123,
    "name":"张三"
  }
}

```
id列表文件： idList.txt
```
123
124
125
```
配置文件  getUserInfo.yaml
```yaml
# 基本参数
method: GET
host: https://abr.test.com
path: /userInfo/getById
# 参数构造语法
getParamTemplateV2: '?id=$0'  #详细构造语法见后文，这里$0表示文件的某行整行
# 源文件
srcFilePath: ./idList.txt     #数据源文件
# 并发控制
workerCoroutineNum: 50       #并发数，最多有多少个并发同时请求
qpsLimit: 100                #限速，每秒最多多少个请求
# 错误检查
resErrNoName: errNo          #错误探针，除了http错误外，还可以检查接口结果中的字段，非0触发错误
# 结果解析
saveResDataExtractor: "data"  # gjson语法，提取结果中data字段，详细结果解析存储配置见后文
```

运行后abr将发送3个httpGet请求
```
GET https://abr.test.com/userInfo/getById?id=123
GET https://abr.test.com/userInfo/getById?id=124
GET https://abr.test.com/userInfo/getById?id=125

```
产1个生名为 ./idList.txt_res 文件，内容如下
```
{"id":123,"name":"张三"}
{"id":124,"name":"张四"}
{"id":125,"name":"张五"}
```



## 配置文件/配置项清单

    配置文件支持json和yaml格式，所有配置项为单层结构。
    以下直接贴配置项对应的的golang结构体，json和yaml配置名见对应tag，其中带星标的为常用配置项。

```go
//1 目标接口静态配置, 帮abr确定一个请求模板
//*1.1 请求的服务地址
Method string `json:"method" yaml:"method"` // GET POST
Host   string `json:"host" yaml:"host"` //  http://domain 、 http://ip:port 
Path   string `json:"path" yaml:"path"` //  需要包含斜杠开头
//1.2 头信息，非必须
Cookies map[string]string `json:"cookies" yaml:"cookies"`
ReqHost string            `json:"reqHost" yaml:"reqHost"`
Header  map[string]string `json:"header" yaml:"header"`


//2 源文件信息，让abr知道如何解析源文件
//*2.1 源文件路径
SrcFilePath  string `json:"srcFilePath" yaml:"srcFilePath"` 
//2.2 只处理源文件的部分行，含义同sql的skip和limit
SrcFileSkip  int    `json:"srcFileSkip" yaml:"srcFileSkip"`  // 默认0，不跳过
//（tips 处理csv文件时可以设置为1跳过标题行）
SrcFileLimit int    `json:"srcFileLimit" yaml:"srcFileLimit"` //  默认0，不限制
//（tips, 可以设为2测试运行2条）
//*2.3 源文件行列结构，搭配参数构造模板使用 （tips 当然源文件也可以每行1个完整参数不构造直接用）
SrcFileColumNum       int    `json:"srcFileColumNum" yaml:"srcFileColumNum"`  //文件列数 默认 1
SrcFileColumSeparator string `json:"srcFileColumSeparator" yaml:"srcFileColumSeparator"` // 列分割符，默认逗号 ","   
//2.4 特殊源文件格式，一般不需要手动指定
SrcFileSepUsRsUs  bool   `json:"srcFileSepUsRsUs" yaml:"srcFileSepUsRsUs"` // 用"\x1E" 当行分隔符， 用 用"\x1F" 当列分隔符, （RS、US当列表分割符的文件，常用来避免分隔符和表格cell内容冲突）
SrcFileLineTrim   string `json:"srcFileLineTrim" yaml:"srcFileLineTrim"` //默认 "\n\r\t"
//2.5 行合并，多行转1行多列 （不影响限速，依然按多行算限速）
MultiLine          int `json:"multiLine" yaml:"multiLine"`  // 默认1，即不合并
MultiLineJoinStr   string  `json:"multiLineJoinStr" yaml:"multiLineJoinStr"`  // 默认 ","  
//tips：比如你有一个单列的id文件，你可以设置multiLine=3 来将3行合并成1行3列，合并后单行如”id1,id2,di3“ ,适用于类似mget等场景参数构造


//3 请求参数构造，给abr一个参数构造模板
// （tips 当前post请求只支持构造json参数，不支持同时构造query和body）  
//*3.1 方式1,不构造，文件每行就是请求参数，文件行格式要求如下
// get请求格式为      ?aa=1&bb=xx ，
// post请求json格式  {"a":1,"b":"bv"}
ParamDirect bool `json:"paramDirect" yaml:"paramDirect"`
//*3.2 get请求 字符串模板参数构造
//参数格式eg   ?aa=$1&bb=$2 (文件一行中第n列替换$n， $0为替换整行)   
GetParamTemplateV2 string `json:"getParamTemplateV2" yaml:"getParamTemplateV2"`
//*3.3 post字符串模板构造
PostParamTemplateV2 string `json:"postParamTemplateV2" yaml:"postParamTemplateV2"` 
// eg {"a":$1,"a2":xxx_$1,"b":"$2","c":$.JSON3}  (文件一行中n列替换$n， $0为整行, $.JSON3 表示将 $3 json编码后替换)
//(tips 通常数字用$n ，简单字符串用"$n" ， 需要转义的字符串用$.JSONn(会自动带上括号)
//3.4 post复杂参数构造
//见后文，提供内嵌函数，js函数，可执行文件等构造请求参数。


//*4 请求限速、限并发
QpsLimit   int `json:"qpsLimit" yaml:"qpsLimit"`   							 // 最大限速
QPerTimeRange   int `json:"qPerTimeRange" yaml:"qPerTimeRange"`  // 限速间隔，单位秒，默认1s ，即n q/1s，适用 < 1qps的限速
WorkerCoroutineNum  int `json:"workerCoroutineNum" yaml:"workerCoroutineNum"` 
//并发度，默认1串行请求,最多几个请求同时请求服务 严格顺序要求请主动置1，压测可以调大WorkerCoroutineNum，
ExpectReqCostMillisecond int `json:"expectReqCostMillisecond" yaml:"expectReqCostMillisecond"` 
//单次请求期望耗时（毫秒），当不设置并发度，系统自动根据此字段安排并发度
 
//5 错误检查、结果解析 ，让abr知道如何发现异常请求以及保存你想要的请求结果 
// 默认视作http连接出错和Http返回非200状态码为错误，
// 5.1 丢弃结果不检查，适合压测提升发压端性能（依旧会检查http错误）
DiscardResBody bool `json:"discardResBody" yaml:"discardResBody"` 
//* 5.2 接口返回结果错误码探针（错误码字段名）
ResErrNoName   string `json:"resErrNoName" yaml:"resErrNoName"`   
//  有该字段则额外解析返回结果中的错误码，非0为错 eg  errno、 errNo
//* 5.3 结果解析，存储
SaveRes              bool   `json:"saveRes" yaml:"saveRes"`  // 保存返回值，默认不保存
SaveResDataExtractor string `json:"saveResDataExtractor" yaml:"saveResDataExtractor"` //默认空，即保存完整的返回的内容
// gjson中json路径格式,解析并保存返回值中的部分数据作为结果 
// eg 某接口返回结果为：{"errNo":0,"data":{"a":1}} 
// saveResDataExtractor设为 data取出其中的'{"a":1}' ; 设为data.a取出 1
SaveResDataTemplate   string   `json:"saveResDataTemplate" yaml:"saveResDataTemplate"`  // 存结果构造模板，
// saveResDataTemplate默认为空，即SaveResDataExtractor解析出的的结果，
// 支持原文件结构$n和saveResDataExtractor的解析结果$resDataExtract 
// eg，比如源文件有两列，你需要用第二列去请求某个接口，然后将第一列及接口输出组合为两列保存，那么可以设置为"$1,$resDataExtract"
// SaveResDataExtractors []string `json:"saveResDataExtractors" yaml:"saveResDataExtractors"` // 作为行列结构输出  未实现 不常用 todo       

//6 结果、错误文件存储位置
ResFilePath string `json:"resFilePath" yaml:"resFilePath"` // 存结果文件位置 默认为  srcfile路径+".res"
ErrFilePath string `json:"errFilePath" yaml:"errFilePath"` // 错误文件位置   默认为  srcfile路径+".err"
// 注意请求失败错误码非0等不合预期的文件数据行都会自动写到err文件，错误信息本身会输出到标准输出


//7 其它参数
Name string    `json:"name" yaml:"name"`           // 任务名，非必须
DetailLog bool `json:"detailLog" yaml:"detailLog"` //输出每次请求的参数及结果，适合尝试或debug用
Statistic bool `json:"statistic" yaml:"statistic"` //持续输出统计信息，接口耗时 成功率等，适合压测用。
Timeout int    `json:"timeout" yaml:"timeout"` // 请求超时、单位毫秒、 一般不用设
Retry  int     `json:"retry" yaml:"retry"`     // http请求失败重试次数、默认0，不重试 
DryRun  bool   `json:"dryRun" yaml:"dryRun"`  // 只构造并打印请求参数，不实际请求
Proxy   string            `json:"proxy" yaml:"proxy"` //代理url

```



## 其它技巧
1. 测试运行

   可以设置以下参数

   ```
   ## 只运行文件前10条数据，并打印每次请求详细的请求参数和返回结果
   detailLog:true
   srcFileLimit:10
   dryRun:true  # 不会实际发生请求
   ```

   

2. 任务意外中断处理
   当前没有中断继续的机制，一般为任务中断重启后 ，需要根据日志获取已经处理的量（不严格，推荐用 最后一个offset减限速qps ），设置srcFileSkip后跳过已执行的行。

## 高级参数构造与结果解析

### 用户定义钩子函数 

abr还提供了以下钩子使用方式，用户可以添加自定义的钩子来完成复杂参数构造和结果解析。（需要在项目指定位置插入代码并重新编译）

+ **参数构造钩子**

   钩子插入位置： {项目目录}/hooks/paramBuilder/

  ```go
  
  // ParamBuilder  请求参数构造钩子
  //  line为源文件中的1行（不一定是一行，如果你指定了记录分割符且不是\n的话）
  //  param 为返回的构造好的参数，
  //  param支持任意结构体类型，
  //  对于get请求， param 应该是string 类型， 格式例子eg  "?a=123&b=xxx"
  //  对于Post请求，param 支持 任意结构/map 、 json string/[]byte 四种类型
  ///*
  type ParamBuilder func(line string) (param interface{}, err error)
  
  var BuiltInParamBuilderNameMap = map[string]ParamBuilder{"": nil}
  
  func init() {
      // insert your hooks here
      BuiltInParamBuilderNameMap["demo"] = func(line string) (param interface{}, err error) {
          param = map[string]interface{}{
              "a": line,
          }
          return
      }
  
  }
  
  ```

  

+ 参数构造补充钩子 （todo）

  期望能解决的场景： 同一个业务不同接口在请求时可能包含相同的附加参数（比如鉴权参数），且通常难以用字符串模板方式构造。

  可以有一个参数构造补充钩子来完成这些通用或复杂的附加参数构造，然后merge进通过字符串模板构造的参数里。

+ 结果解析钩子  （todo）





### 外置参数构造器 (todo)

 如何能不更改代码不重新编译实现复杂参数构造呢？

 外置可执行脚本？ lua ？  js 函数？（golang 可以通过chrome v8执行JavaScript函数）



## 欢迎一起完善abr

abr核心逻辑是一个生产消费模型(代码位于./core/abr)，代码量仅数百行，没有复杂结构，欢迎直接补充您所需的功能并提交pr。