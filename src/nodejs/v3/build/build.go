package build

import (
	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
)

const NodeDependency = "node"

func CreateLaunchMetadata() libbuildpackV3.LaunchMetadata {
	return libbuildpackV3.LaunchMetadata{
		Processes: libbuildpackV3.Processes{
			libbuildpackV3.Process{
				Type:    "web",
				Command: "npm start",
			},
		},
	}
}

type Node struct {
	launchLayer libjavabuildpack.DependencyLaunchLayer
}

func NewNode(builder libjavabuildpack.Build) (Node, bool, error) {
	bp, ok := builder.BuildPlan[NodeDependency]
	if !ok {
		return Node{}, false, nil
	}

	deps, err := builder.Buildpack.Dependencies()
	if err != nil {
		return Node{}, false, err
	}

	dep, err := deps.Best(NodeDependency, bp.Version, builder.Stack)
	if err != nil {
		return Node{}, false, err
	}

	node := Node{builder.Launch.DependencyLayer(dep)}
	return node, true, nil
}

func (n Node) Contribute() error {
	return n.launchLayer.Contribute(func(artifact string, layer libjavabuildpack.DependencyLaunchLayer) error {
		layer.Logger.SubsequentLine("Expanding to %s", layer.Root)
		if err := libjavabuildpack.ExtractTarGz(artifact, layer.Root, 1); err != nil {
			return err
		}
		return nil
	})
	return nil
}
