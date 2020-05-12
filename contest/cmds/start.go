package cmds

type StartCommand struct {
}

func (cmd *StartCommand) Run() error {
	// load design file

	// create docker env

	// generate genesis block and others
	// BLOCK moves to contest/main.go
	// BLOCK set basic policy
	/*{
		log.Debug().Msg("NodeRunner generated")

		if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), nil); err != nil {
			log.Error().Err(err).Msg("failed to create genesis block generator")

			os.Exit(1)
		} else if blk, err := gg.Generate(); err != nil {
			log.Error().Err(err).Msg("failed to generate genesis block")

			os.Exit(1)
		} else {
			log.Info().Interface("block", blk).Msg("genesis block created")
		}
	}*/

	// distribute blocks to other nodes

	// start nodes

	return nil
}
