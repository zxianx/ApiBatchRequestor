package core

import "apiBatchRequester/hooks/paramBuilder"

type ApiPosterConf struct {
    Name string `json:"name" yaml:"name"`

    // 请求的服务地址
    Host    string `json:"host" yaml:"host"` //  http://xxx.cc ,  http://192.xx.xx.xx:8080 , svc额外需要ship那个开发工具
    Path    string `json:"path" yaml:"path"` //  斜杠开头
    Method  string `json:"method" yaml:"method"`
    Timeout int    `json:"timeout" yaml:"timeout"`
    Retry   int    `json:"retry" yaml:"retry"`   // http请求失败重试次数、默认0，不重试
    DryRun  bool   `json:"dryRun" yaml:"dryRun"` // 只构造并打印请求参数，不实际请求

    // 源文件信息
    SrcFilePath  string `json:"srcFilePath" yaml:"srcFilePath"`
    SrcFileSkip  int    `json:"srcFileSkip" yaml:"srcFileSkip"`   // 含义同sql的skip和limit
    SrcFileLimit int    `json:"srcFileLimit" yaml:"srcFileLimit"` // 0为不限制
    //选择字符串模板请求参数构造中时需要列信息
    SrcFileColumNum       int    `json:"srcFileColumNum" yaml:"srcFileColumNum"`             // default 1
    SrcFileColumSeparator string `json:"srcFileColumSeparator" yaml:"srcFileColumSeparator"` // default string
    srcFileLineSep        byte

    // 特殊源文件格式
    SrcFileSepUsRsUs bool   `json:"srcFileSepUsRsUs" yaml:"srcFileSepUsRsUs"` // 用"\x1E" 当行分隔符， 用 用"\x1F" 当列分隔符
    SrcFileLineTrim  string `json:"srcFileLineTrim" yaml:"srcFileLineTrim"`   // 默认 "\n\r\t"

    // 请求参数构造,选一个构造方式 （暂不支持同时构造url和body）

    //不构造，文件每行就是请求参数，文件行格式要求，get请求格式 【?aa=1&bb=xx】，post json格式【 {"a":1,"b":"bv"} 】
    ParamDirect bool `json:"paramDirect" yaml:"paramDirect"`

    //get 字符串模板，适合简单参数构造
    GetParamTemplate   string `json:"getParamTemplate" yaml:"getParamTemplate"`     //参数格式eg   ?aa=%s&bb=%s (文件一行中列依次替换%s)
    GetParamTemplateV2 string `json:"getParamTemplateV2" yaml:"getParamTemplateV2"` //参数格式eg   ?aa=$1&bb=$2 (文件一行中n列替换$n， $0为整行)
    // post
    //字符串模板，适合简单参数构造，
    PostParamTemplate   string `json:"postParamTemplate" yaml:"postParamTemplate"`     // eg   {"a":%s,"b":"%s"}   (文件一行中列依次替换%s)
    PostParamTemplateV2 string `json:"postParamTemplateV2" yaml:"postParamTemplateV2"` // eg   {"a":$1,"a2":xx_$1,"b":"$2"}  (文件一行中n列替换$n， $0为整行)

    //函数 postParamBuilderFunc   接受1个字符串参数,文件行内容，todo
    BuiltInParamBuilderName string                    `json:"builtInParamBuilderName" yaml:"builtInParamBuilderName"`
    BuiltInParamBuilder     paramBuilder.ParamBuilder `json:"-" yaml:"-"`

    ParamBuilderJs      string `json:"paramBuilderJs" yaml:"paramBuilderJs"`
    ParamBuilderExecCmd string `json:"paramBuilderExecCmd" yaml:"paramBuilderExecCmd"`
    // ParamBuilderExtInfoAppenderName string
    // PreParamBuilderTransferName     string

    MultiLine        int    `json:"multiLine" yaml:"multiLine"`               //搭配post高级参数构造， 一次发送多行数据发送给postParamBuilderFunc
    MultiLineJoinStr string `json:"multiLineJoinStr" yaml:"multiLineJoinStr"` // 配置MultiLine时候，行分割符替换的符号 ，默认 ","

    QpsLimit                 int `json:"qpsLimit" yaml:"qpsLimit"`                                 // 最大限速
    QPerTimeRange            int `json:"qPerTimeRange" yaml:"qPerTimeRange"`                       // 秒，默认1 ，即n q/1s，适用 < 1qps的限速
    WorkerCoroutineNum       int `json:"workerCoroutineNum" yaml:"workerCoroutineNum"`             //默认1  并发度,最多几个请求同时请求服务 严格顺序要求请主动置1，压测可以调大WorkerCoroutineNum
    ExpectReqCostMillisecond int `json:"expectReqCostMillisecond" yaml:"expectReqCostMillisecond"` // 单次请求期望耗时（毫秒），当不设置并发度，系统自动根据此字段安排并发度

    DiscardResBody bool `json:"discardResBody" yaml:"discardResBody"` //丢弃返回body，不处理

    ResErrNoName string `json:"resErrNoName" yaml:"resErrNoName"` // 默认不解析body中的errNo， eg  errno、 errNo

    SaveRes               bool     `json:"saveRes" yaml:"saveRes"`                             // 保存返回值
    SaveResDataExtractor  string   `json:"saveResDataExtractor" yaml:"saveResDataExtractor"`   // gjson 格式，解析并保存返回值中的部分数据作为整行输出么，空为结果全部保存
    SaveResDataTemplate   string   `json:"saveResDataTemplate" yaml:"saveResDataTemplate"`     // 存结果构造模板，默认为 SaveResDataExtractor 的结果，支持  $n $resDataExtract ， eg "$0,$resDataExtract"
    SaveResDataExtractors []string `json:"saveResDataExtractors" yaml:"saveResDataExtractors"` // 作为行列结构输出 tod0

    ResFilePath string `json:"resFilePath" yaml:"resFilePath"` //  default srcfile+后缀 .res
    ErrFilePath string `json:"errFilePath" yaml:"errFilePath"` //  default srcfile+后缀 .err

    Cookies map[string]string `json:"cookies" yaml:"cookies"`
    ReqHost string            `json:"reqHost" yaml:"reqHost"`
    Proxy   string            `json:"proxy" yaml:"proxy"`
    Header  map[string]string `json:"header" yaml:"header"`

    DetailLog bool `json:"detailLog" yaml:"detailLog"` //输出每次请求及结果，debug用

    Statistic bool `json:"statistic" yaml:"statistic"`
}
