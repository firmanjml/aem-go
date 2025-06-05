package manager

type DownloadExtension interface {
	ListVersions(version *string) ([]string, error)
	CheckVersion(version string) (bool, error)
	GetDownloadURL(version string) (string, error)
}

type BaseExtension struct {
	BaseUrl string
}

type ExtensionManager struct {
	extensions map[string]DownloadExtension
}

func NewExtensionManager() *ExtensionManager {
	return &ExtensionManager{
		extensions: make(map[string]DownloadExtension),
	}
}

func (em *ExtensionManager) RegisterExtension(name string, ext DownloadExtension) {
	em.extensions[name] = ext
}

func (em *ExtensionManager) GetExtension(name string) (DownloadExtension, bool) {
	ext, exists := em.extensions[name]
	return ext, exists
}

func (em *ExtensionManager) ListExtensions() []string {
	names := make([]string, 0, len(em.extensions))
	for name := range em.extensions {
		names = append(names, name)
	}
	return names
}
