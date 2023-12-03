package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/nexablock/nexablock/x/vesting/client/cli"
)

var RegisterClawbackProposalHandler = govclient.NewProposalHandler(cli.NewClawbackProposalCmd)
