/*
	Copyright 2017 by Christian Hüning (christianhuening@googlemail.com).

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at
		http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package k8sclient

import (
	"fmt"
	"testing"
)

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
	if err != nil {
		t.Error(err)
	}
	if k1 != "uP_uP-Chief" {
		t.Error("Incorrect replacement of '/'")
	}

	g2 := "u_P/uP-Chief"
	k2, err := GitlabNameToK8sLabel(g2)
	if err != nil {
		t.Error(err)
	}
	if k2 != "u__P_uP-Chief" {
		t.Error("Incorrect replacement of '/'")
	}

	g3 := "u__.P/uP-Chief"
	k3, err := GitlabNameToK8sLabel(g3)
	if err != nil {
		t.Error(err)
	}
	if k3 != "u____.P_uP-Chief" {
		t.Error("Incorrect replacement of '/'. Res: " + k3)
	}

	g4 := "uP-uP-Chief"
	k4, err := GitlabNameToK8sLabel(g4)
	if err != nil {
		t.Error(err)
	}
	if k4 != "uP-uP-Chief" {
		t.Error("Incorrect replacement of nothing")
	}
}

func TestK8sLabelToGitlabName(t *testing.T) {
	k1 := "uP_uP-Chief"
	g1, err := K8sLabelToGitlabName(k1)
	if err != nil {
		t.Error(err)
	}
	if g1 != "uP/uP-Chief" {
		t.Error("Incorrect replacement of '/'. Res: " + g1)
	}

	k2 := "u__P_uP-Chief"
	g2, err := K8sLabelToGitlabName(k2)
	if err != nil {
		t.Error(err)
	}
	if g2 != "u_P/uP-Chief" {
		t.Error("Incorrect replacement of '/'. Res: " + g2)
	}

	k3 := "u____.P_uP-Chief"
	g3, err := K8sLabelToGitlabName(k3)
	if err != nil {
		t.Error(err)
	}
	if g3 != "u__.P/uP-Chief" {
		t.Error("Incorrect replacement of '/'. Res: " + g3)
	}

	k4 := "uP-uP-Chief"
	g4, err := K8sLabelToGitlabName(k4)
	if err != nil {
		t.Error(err)
	}
	if g4 != "uP-uP-Chief" {
		t.Error("Incorrect replacement of '/'. Res: " + g4)
	}
}

func TestGetGroupRoleName(t *testing.T) {
	masterRole := GetGroupRoleName("Master")
	if masterRole != "gitlab-group-master" {
		t.Error(fmt.Sprintf("Expected gitlab-group-master , but was %s", masterRole))
	}
}

func TestGetProjectRoleName(t *testing.T) {
	masterRole := GetProjectRoleName("Master")
	if masterRole != "gitlab-project-master" {
		t.Error(fmt.Sprintf("Expected gitlab-project-master , but was %s", masterRole))
	}
}
