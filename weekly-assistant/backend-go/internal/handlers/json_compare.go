package handlers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

type diffItem struct {
	Path     string      `json:"path"`
	Type     string      `json:"type"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

func JsonCompare(c *gin.Context) {
	var req struct {
		JsonA string `json:"json_a"`
		JsonB string `json:"json_b"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}
	if req.JsonA == "" || req.JsonB == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "两个 JSON 都不能为空")
		return
	}

	var a, b interface{}
	if err := json.Unmarshal([]byte(req.JsonA), &a); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "JSON A 解析失败: "+err.Error())
		return
	}
	if err := json.Unmarshal([]byte(req.JsonB), &b); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "JSON B 解析失败: "+err.Error())
		return
	}

	var diffs []diffItem
	compare("$", a, b, &diffs)

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})

	utils.Success(c, gin.H{
		"differences": diffs,
		"match":       len(diffs) == 0,
	})
}

func compare(path string, a, b interface{}, diffs *[]diffItem) {
	if reflect.DeepEqual(a, b) {
		return
	}

	if a == nil && b != nil {
		*diffs = append(*diffs, diffItem{Path: path, Type: "added", NewValue: b})
		return
	}
	if a != nil && b == nil {
		*diffs = append(*diffs, diffItem{Path: path, Type: "removed", OldValue: a})
		return
	}

	am, aIsMap := a.(map[string]interface{})
	bm, bIsMap := b.(map[string]interface{})
	if aIsMap && bIsMap {
		keys := map[string]bool{}
		for k := range am {
			keys[k] = true
		}
		for k := range bm {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			childPath := path + "." + k
			va, aHas := am[k]
			vb, bHas := bm[k]
			if aHas && !bHas {
				*diffs = append(*diffs, diffItem{Path: childPath, Type: "removed", OldValue: va})
			} else if !aHas && bHas {
				*diffs = append(*diffs, diffItem{Path: childPath, Type: "added", NewValue: vb})
			} else {
				compare(childPath, va, vb, diffs)
			}
		}
		return
	}

	al, aIsList := a.([]interface{})
	bl, bIsList := b.([]interface{})
	if aIsList && bIsList {
		maxLen := len(al)
		if len(bl) > maxLen {
			maxLen = len(bl)
		}
		for i := 0; i < maxLen; i++ {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if i >= len(al) {
				*diffs = append(*diffs, diffItem{Path: childPath, Type: "added", NewValue: bl[i]})
			} else if i >= len(bl) {
				*diffs = append(*diffs, diffItem{Path: childPath, Type: "removed", OldValue: al[i]})
			} else {
				compare(childPath, al[i], bl[i], diffs)
			}
		}
		return
	}

	*diffs = append(*diffs, diffItem{Path: path, Type: "changed", OldValue: a, NewValue: b})
}
