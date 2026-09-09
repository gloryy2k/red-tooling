package config

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	once    sync.Once
	rtHome  string
)

// Home returns the RT home directory (~/.rt).
func Home() string {
	once.Do(func() {
		if env := os.Getenv("RT_HOME"); env != "" {
			rtHome = env
		} else {
			home, _ := os.UserHomeDir()
			rtHome = filepath.Join(home, ".rt")
		}
	})
	return rtHome
}

// DataDir returns the directory where engagement databases live.
func DataDir() string {
	return filepath.Join(Home(), "data")
}

// KeysDir returns the directory for operator keys.
func KeysDir() string {
	return filepath.Join(Home(), "keys")
}

// TLSDir returns the directory for TLS certificates.
func TLSDir() string {
	return filepath.Join(Home(), "tls")
}

// EnsureDirs creates all necessary RT directories.
func EnsureDirs() error {
	dirs := []string{
		Home(),
		DataDir(),
		KeysDir(),
		filepath.Join(KeysDir(), "peers"),
		TLSDir(),
		filepath.Join(Home(), "playbooks"),
		filepath.Join(Home(), "checklists"),
		filepath.Join(Home(), "templates"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}
	return nil
}

// ActiveEngagementFile returns the path to the file that stores the active engagement name.
func ActiveEngagementFile() string {
	return filepath.Join(Home(), "active")
}

// GetActiveEngagement reads the currently active engagement name.
func GetActiveEngagement() string {
	data, err := os.ReadFile(ActiveEngagementFile())
	if err != nil {
		return ""
	}
	return string(data)
}

// SetActiveEngagement writes the active engagement name.
func SetActiveEngagement(name string) error {
	return os.WriteFile(ActiveEngagementFile(), []byte(name), 0600)
}

// EncryptedDBPath returns the path to the encrypted database file.
func EncryptedDBPath(engagementName string) string {
	return filepath.Join(DataDir(), engagementName+".db.enc")
}

// PlainDBPath returns the path to the decrypted (active) database file.
func PlainDBPath(engagementName string) string {
	return filepath.Join(DataDir(), engagementName+".db")
}
