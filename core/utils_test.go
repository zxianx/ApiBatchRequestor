package core

import (
    "github.com/stretchr/testify/assert"
    "testing"
)

func Test_appendParamsToURL(t *testing.T) {
    append := map[string]interface{}{
        "a": 1,
        "b": "bb",
        "c": true,
    }
    res1, _ := appendParamsToURL(append, "localhost:8080/get?base=1")
    assert.Equal(t, res1, "localhost:8080/get?a=1&b=bb&base=1&c=true")
}
