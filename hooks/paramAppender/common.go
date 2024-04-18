package paramAppender

import "time"

// ParamAppender  参数构造补充钩子， 支持get和post请求的参数补充
//  line为源文件中的1行（不一定是一行，如果你指定了记录分割符且不是\n的话）
//  append 为返回的补充构造参数，如果是get请求则拼接到原始参数后面，post请求则merge到请求参数的最外层
//  append 内部字段不支数组及嵌套结构
///*
type ParamAppender func(line string) (param map[string]interface{}, err error)

var BuiltInParamAppenderNameMap = map[string]ParamAppender{"": nil}

func init() {
    // insert your hooks here

    //为参数补充一个名为time的时间戳
    BuiltInParamAppenderNameMap["time"] = func(line string) (append map[string]interface{}, err error) {
        append = map[string]interface{}{
            "time": time.Now().Unix(),
        }
        return
    }

}
