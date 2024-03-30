package paramBuilder

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
