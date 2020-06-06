package cmds

type NodeCommand struct {
	NodeInfo  NodeInfoCommand  `cmd:"" help:"get node info from mitum node"`
	Manifests ManifestsCommand `cmd:"" help:"get manifests from mitum node"`
	Blocks    BlocksCommand    `cmd:"" help:"get blocks from mitum node"`
}
