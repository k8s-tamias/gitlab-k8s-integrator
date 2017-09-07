package k8sclient

import "testing"

/*
Rules:
1) “.” <-> “.”
2) “-” <-> “-”
3) “_” <-> “__”
4) “/” <-> “_”
 */

func TestGitlabNameToK8sLabel(t *testing.T) {
	g1 := "uP/uP-Chief"
	k1, err := GitlabNameToK8sLabel(g1)
	if err != nil {t.Error(err)}
	if k1 != "uP_uP-Chief" { t.Error("Incorrect replacement of '/'")}

	g2 := "u_P/uP-Chief"
	k2, err := GitlabNameToK8sLabel(g2)
	if err != nil {t.Error(err)}
	if k2 != "u__P_uP-Chief" { t.Error("Incorrect replacement of '/'")}

	g3 := "u__.P/uP-Chief"
	k3, err := GitlabNameToK8sLabel(g3)
	if err != nil {t.Error(err)}
	if k3 != "u____.P_uP-Chief" { t.Error("Incorrect replacement of '/'. Res: "+ k3)}
}

func TestK8sLabelToGitlabName(t *testing.T) {
	k1 := "uP_uP-Chief"
	g1, err := K8sLabelToGitlabName(k1)
	if err != nil {t.Error(err)}
	if g1 != "uP/uP-Chief" { t.Error("Incorrect replacement of '/'. Res: "+ g1)}
	
	k2 := "u__P_uP-Chief"
	g2, err := K8sLabelToGitlabName(k2)
	if err != nil {t.Error(err)}
	if g2 != "u_P/uP-Chief" { t.Error("Incorrect replacement of '/'. Res: "+ g2)}

	k3 := "u____.P_uP-Chief"
	g3, err := K8sLabelToGitlabName(k3)
	if err != nil {t.Error(err)}
	if g3 != "u__.P/uP-Chief" { t.Error("Incorrect replacement of '/'. Res: "+ g3)}
}