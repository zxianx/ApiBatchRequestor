package core

import (
    "fmt"
    "testing"
)

func TestTemplateReplace(t *testing.T) {
    fmt.Println(TemplateReplace("x$2 $0*", "aaa;bbb;ccc", ";"))
    fmt.Println(TemplateReplace("x$2,$3*", "aaa;bbb;ccc", ";"))
    fmt.Println(TemplateReplace("x$4 $0*", "aaa;bbb;ccc", ";"))
    fmt.Println(TemplateReplace("_$1_$JSON2_", "aa\na;b\"b\nb;ccc", ";"))
}
