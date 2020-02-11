package tammany

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"golang.org/x/net/context"
)

func (g *Game) cancelFinish(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validateCancelFinish(ctx); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
	} else {
		g.SubPhase = noSubPhase
		tmpl, act = "tammany/flash_notice", game.UndoPop
	}
	return
}

func (g *Game) validateCancelFinish(ctx context.Context) (err error) {
	switch {
	case !g.CUserIsCPlayerOrAdmin(ctx):
		err = sn.NewVError("Only the current player can take this action.")
	case !g.inActionPhase():
		err = sn.NewVError("Wrong phase for performing this action.")
	}
	return
}
