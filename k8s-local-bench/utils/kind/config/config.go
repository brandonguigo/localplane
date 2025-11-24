package kindconfig

import (
	"os"

	"go.yaml.in/yaml/v3"
)

type KindCluster struct {
	Kind       string     `yaml:"kind"`
	APIVersion string     `yaml:"apiVersion"`
	Nodes      []KindNode `yaml:"nodes"`
}

type KindNode struct {
	Role        string       `yaml:"role"`
	ExtraMounts []ExtraMount `yaml:"extraMounts,omitempty"`
}

type ExtraMount struct {
	HostPath      string `yaml:"hostPath"`
	ContainerPath string `yaml:"containerPath"`
}

func LoadKindConfig(path string) (*KindCluster, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &KindCluster{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func AddExtraMount(cfg *KindCluster, host, container string) {
	for i := range cfg.Nodes {
		cfg.Nodes[i].ExtraMounts = append(cfg.Nodes[i].ExtraMounts, ExtraMount{
			HostPath:      host,
			ContainerPath: container,
		})
	}
}

func SaveKindConfig(path string, cfg *KindCluster) error {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}
