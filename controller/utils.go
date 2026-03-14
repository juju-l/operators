package controller

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"tst-operator/apis/example.com/v1beta1"
)

// PatchTstStatus 使用 MergePatch 安全更新 Status
// rv: 用于校验资源是否被外部篡改
func (c *Controller) PatchTstStatus(
	ns, name, rv string,
	newStatus *v1beta1.TstStatus,
) error {
	// 1. 获取最新对象（再次校验 RV）
	current, err := c.clientset.Tsts(ns).Get(context.TODO(),name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if current.ResourceVersion != rv {
		return fmt.Errorf("resource changed, rv mismatch: %s vs %s", rv, current.ResourceVersion)
	}

	// 2. 构造 Patch
	// oldData, _ := json.Marshal(current.Status)
	// newData, _ := json.Marshal(newStatus)
	// patch, err := jsonpatch.CreateMergePatch(oldData, newData)
	// ================= 只改这3行 =================
	oldData, _ := json.Marshal(map[string]any{"status": current.Status})
	newData, _ := json.Marshal(map[string]any{"status": newStatus})
	patch, err := jsonpatch.CreateMergePatch(oldData, newData)
	// ==============================================
	if err != nil {
		return err
	}

	// 3. 执行 Patch（仅更新 status）
	_, err = c.clientset.Tsts(ns).Patch(context.TODO(),
		name,
		types.MergePatchType,
		patch,
		metav1.PatchOptions{},
		"status", // subresource: status
	)
	return err
}