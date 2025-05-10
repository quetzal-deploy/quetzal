package ssh

import (
	"testing"

	"github.com/quetzal-deploy/quetzal/internal/common"
)

func TestCreateSSHContextWithoutEnvVars(t *testing.T) {
	sshContext := CreateSSHContext(&common.QuetzalOptions{})

	if sshContext.SshOptions.ConfigFile != "" {
		t.Fatalf("Expected empty SSH ConfigFile, got %s", sshContext.SshOptions.ConfigFile)
	}

	if sshContext.SshOptions.DefaultUsername != "" {
		t.Fatalf("Expected empty DefaultUsername, got %s", sshContext.SshOptions.DefaultUsername)
	}

	if sshContext.SshOptions.IdentityFile != "" {
		t.Fatalf("Expected empty IdentityFile, got %s", sshContext.SshOptions.IdentityFile)
	}

	if sshContext.SshOptions.SkipHostKeyCheck != false {
		t.Fatalf("Expected SkipHostKeyCheck = false")
	}
}

func TestCreateSSHContextWithEnvVars(t *testing.T) {
	defaultUsername := "quetzal"
	t.Setenv("SSH_USER", defaultUsername)

	skipHostKeyCheck := "YES"
	t.Setenv("SSH_SKIP_HOST_KEY_CHECK", skipHostKeyCheck)

	sshConfigFile := "~/.ssh/config"
	t.Setenv("SSH_CONFIG_FILE", sshConfigFile)

	sshIdentityFile := "/fake/file"
	t.Setenv("SSH_IDENTITY_FILE", sshIdentityFile)

	sshContext := CreateSSHContext(&common.QuetzalOptions{})

	if sshContext.SshOptions.ConfigFile != sshConfigFile {
		t.Fatalf("Expected SSH ConfigFile = '%s', got '%s'", sshConfigFile, sshContext.SshOptions.ConfigFile)
	}

	if sshContext.SshOptions.DefaultUsername != defaultUsername {
		t.Fatalf("Expected SSH DefaultUsername = '%s', got '%s'", defaultUsername, sshContext.SshOptions.DefaultUsername)
	}

	if sshContext.SshOptions.IdentityFile != sshIdentityFile {
		t.Fatalf("Expected IdentityFile = '%s', got '%s'", sshIdentityFile, sshContext.SshOptions.IdentityFile)
	}

	if sshContext.SshOptions.SkipHostKeyCheck != true {
		t.Fatalf("Expected SkipHostKeyCheck = true")
	}
}

func TestCreateSSHContextSkipHostKeyCheck(t *testing.T) {
	skipTrueValues := []string{"YES", "yES", "yes", "TRUE", "trUe", "true"}
	skipFalseValues := []string{"NO", "nO", "no", "n0", "FALSE", "FaLse", "false", "smørrebrød"}

	for _, input := range skipTrueValues {
		t.Setenv("SSH_SKIP_HOST_KEY_CHECK", input)

		sshContext := CreateSSHContext(&common.QuetzalOptions{})

		if sshContext.SshOptions.SkipHostKeyCheck != true {
			t.Fatalf("Expected SkipHostKeyCheck = true using SSH_SKIP_HOST_KEY_CHECK='%s'", input)
		}
	}

	for _, input := range skipFalseValues {
		t.Setenv("SSH_SKIP_HOST_KEY_CHECK", input)

		sshContext := CreateSSHContext(&common.QuetzalOptions{})

		if sshContext.SshOptions.SkipHostKeyCheck != false {
			t.Fatalf("Expected SkipHostKeyCheck = false using SSH_SKIP_HOST_KEY_CHECK='%s'", input)
		}
	}
}
