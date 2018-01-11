package graylog
/*
import (
	"os"
	"testing"
)

const namespaceName = "test-18F81907-7C88-438E-A648-2069EF5B95B2"
const userName = "testaccount"

func TestMain(m *testing.M) {

	os.Setenv("GRAYLOG_BASE_URL", "")
	os.Setenv("GRAYLOG_ADMIN_USER", "")
	os.Setenv("GRAYLOG_ADMIN_PASSWORD", "")

	ret := m.Run()
	os.Exit(ret)
}

func TestCreateStream(t *testing.T) {
	done, _ := CreateStream(namespaceName)
	if !done {
		t.Fail()
	}
	done2, _ := CreateStream(namespaceName)
	if !done2 {
		t.Fail()
	}
	DeleteStream(namespaceName)
}

func TestIsStreamAlreadyPresent(t *testing.T) {
	isCreated, _ := isStreamAlreadyCreated(namespaceName)
	if isCreated {
		t.Fail()
	}
	done, _ := CreateStream(namespaceName)
	if !done {
		t.Fail()
	}
	isCreated, _ = isStreamAlreadyCreated(namespaceName)
	if !isCreated {
		t.Fail()
	}
	DeleteStream(namespaceName)
	isCreated, _ = isStreamAlreadyCreated(namespaceName)
	if isCreated {
		t.Fail()
	}

}

func TestGetStreamByNamespaceName(t *testing.T) {
	done, _ := CreateStream(namespaceName)
	if !done {
		t.Fail()
	}
	str, err := getStreamByNamespaceName(namespaceName)
	if err != nil {
		t.Error(err)
	}
	if str.Title != namespaceName {
		t.Fail()
	}
	DeleteStream(namespaceName)
}

func TestDeleteStream(t *testing.T) {
	done, _ := CreateStream(namespaceName)
	if !done {
		t.Fail()
	}
	DeleteStream(namespaceName)
	if cond, _ := isStreamAlreadyCreated(namespaceName); cond == true {
		t.Fail()
	}
}

func TestGrandAndTakePermissionForStream(t *testing.T) {
	done, _ := CreateStream(namespaceName)
	if !done {
		t.Fail()
	}
	success := GrantPermissionForStream(namespaceName, userName)
	if !success {
		t.Fail()
	}
	succesTake := TakePermissionForStream(namespaceName, userName)
	if !succesTake {
		t.Fail()
	}
	DeleteStream(namespaceName)
}

func TestGetGraylogUser(t *testing.T) {
	// must not fail
	user, err := getGraylogUser("admin")
	if err != nil {
		t.Fail()
	}
	if user == nil || user.Username != "admin" {
		t.Fail()
	}

	// must fail
	user2, err2 := getGraylogUser("UserUnknown1337")
	if err2 == nil || user2 != nil{
		t.Fail()
	}

}
*/