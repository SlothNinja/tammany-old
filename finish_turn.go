package tammany

import (
	"net/http"
	"time"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user/stats"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

func finish(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))

		g := gameFrom(ctx)
		oldCP := g.CurrentPlayer()

		var (
			s   *stats.Stats
			cs  contest.Contests
			err error
		)

		switch g.Phase {
		case actions:
			s, cs, err = g.actionsPhaseFinishTurn(ctx)
		case placeImmigrant:
			s, cs, err = g.placeImmigrantPhaseFinishTurn(ctx)
		case takeFavorChip:
			s, cs, err = g.takeChipPhaseFinishTurn(ctx)
		case elections:
			s, cs, err = g.electionPhaseFinishTurn(ctx)
		case assignCityOffices:
			s, err = g.assignOfficesPhaseFinishTurn(ctx)
		default:
			err = sn.NewVError("Improper Phase for finishing turn.")
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
			return
		}

		// cs != nil then game over
		if cs != nil {
			g.Phase = gameOver
			g.Status = game.Completed
			if err = g.save(ctx, wrap(s.GetUpdate(ctx, time.Time(g.UpdatedAt)), cs)...); err == nil {
				err = g.sendEndGameNotifications(ctx)
			}
		} else {
			if err = g.save(ctx, s.GetUpdate(ctx, time.Time(g.UpdatedAt))); err == nil {
				if newCP := g.CurrentPlayer(); newCP != nil && oldCP.ID() != newCP.ID() {
					err = g.SendTurnNotificationsTo(ctx, newCP)
				}
			}
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
		}

		return
	}
	return
}

func (g *Game) validateFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var cp *Player

	switch cp, s = g.CurrentPlayer(), stats.Fetched(ctx); {
	case s == nil:
		err = sn.NewVError("missing stats for player.")
	case !g.CUserIsCPlayerOrAdmin(ctx):
		err = sn.NewVError("Only the current player may finish a turn.")
	case !cp.PerformedAction:
		err = sn.NewVError("%s has yet to perform an action.", g.NameFor(cp))
	case g.ImmigrantInTransit != noNationality:
		err = sn.NewVError("You must complete move of %s immigrant before finishing turn.", g.ImmigrantInTransit)
	}
	return
}

// ps is an optional parameter.
// If no player is provided, assume current player.
func (g *Game) nextPlayer(ps ...game.Playerer) *Player {
	if nper := g.NextPlayerer(ps...); nper != nil {
		return nper.(*Player)
	}
	return nil
}

func (g *Game) actionsPhaseFinishTurn(ctx context.Context) (s *stats.Stats, cs contest.Contests, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if s, err = g.validateActionsPhaseFinishTurn(ctx); err != nil {
		return
	}

	cp := g.CurrentPlayer()
	if g.CanUseOffice(cp) && restful.GinFrom(ctx).PostForm("action") != "confirm-finish" {
		g.SubPhase = officeWarning
		err = g.cache(ctx)
		return
	}

	restful.AddNoticef(ctx, "%s finished turn.", g.NameFor(cp))

	np := g.nextPlayer()
	g.beginningOfTurnResetFor(np)
	g.setCurrentPlayers(np)

	if game.IndexFor(np, g.Playerers) == 0 {
		switch g.Year() {
		case 4, 8, 12, 16:
			// if cs != nil then end game
			if cs = g.startElections(ctx); cs != nil {
				return
			}
		default:
			g.setYear(g.Year() + 1)
		}
	}

	if g.Phase == actions {
		g.castleGardenPhase()
	}

	return
}

func (g *Game) validateActionsPhaseFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	if s, err = g.validateFinishTurn(ctx); g.Phase != actions {
		err = sn.NewVError(`Expected "Actions" phase but have %q phase.`, g.Phase)
	}
	return
}

func (g *Game) electionPhaseFinishTurn(ctx context.Context) (s *stats.Stats, cs contest.Contests, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if s, err = g.validateElectionPhaseFinishTurn(ctx); err != nil {
		return
	}

	restful.AddNoticef(ctx, "%s finished turn.", g.NameFor(g.CurrentPlayer()))

	cs = g.continueElections(ctx)
	return
}

func (g *Game) continueElections(ctx context.Context) (cs contest.Contests) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	// when true election phase is over
	if !g.electionsTillUnresolved(ctx) {
		return
	}

	g.startAwardChipsPhase(ctx)
	g.startScoreVictoryPointsPhase(ctx)
	g.newTurnOrder(ctx)

	cs = g.startCityOfficesPhase(ctx)
	return
}

func (g *Game) validateElectionPhaseFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	if s, err = g.validateFinishTurn(ctx); g.Phase != elections {
		err = sn.NewVError(`Expected "Elections" phase but have %q phase.`, g.Phase)
	}
	return
}

func (g *Game) electionsTillUnresolved(ctx context.Context) (done bool) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	for _, w := range g.ActiveWards() {
		if !w.Resolved {
			if g.CurrentWard() == w {
				if !g.resolve(ctx, w) {
					return
				}
			} else {
				if !g.startElectionIn(ctx, w) {
					return
				}
			}
		}
	}
	done = true
	return
}

func (g *Game) placeImmigrantPhaseFinishTurn(ctx context.Context) (s *stats.Stats, cs contest.Contests, err error) {
	if s, err = g.validatePlaceImmigrantPhaseFinishTurn(ctx); err != nil {
		return
	}

	g.Phase = elections
	g.CurrentWard().Resolved = true
	cs = g.continueElections(ctx)
	return
}

func (g *Game) validatePlaceImmigrantPhaseFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	if s, err = g.validateFinishTurn(ctx); g.Phase != placeImmigrant {
		err = sn.NewVError(`Expected "Place Immigrant" phase but have %q phase.`, g.Phase)
	}
	return
}

func (g *Game) takeChipPhaseFinishTurn(ctx context.Context) (s *stats.Stats, cs contest.Contests, err error) {
	if s, err = g.validateTakeChipPhaseFinishTurn(ctx); err == nil {
		g.Phase = elections
		g.CurrentWard().Resolved = true
		cs = g.continueElections(ctx)
	}
	return
}

func (g *Game) validateTakeChipPhaseFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	if s, err = g.validateFinishTurn(ctx); g.Phase != takeFavorChip {
		err = sn.NewVError(`Expected "Take Favor Chip" phase but have %q phase.`, g.Phase)
	}
	return
}

func (g *Game) assignOfficesPhaseFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	if s, err = g.validateAssignOfficesPhaseFinishTurn(ctx); err == nil {
		restful.AddNoticef(ctx, "%s finished turn.", g.NameFor(g.CurrentPlayer()))

		g.startNextTerm()
	}
	return
}

func (g *Game) validateAssignOfficesPhaseFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	switch s, err = g.validateFinishTurn(ctx); {
	case err != nil:
	case g.Phase != assignCityOffices:
		err = sn.NewVError(`Expected "Assign City Offices" phase but have %q phase.`, g.Phase)
	case !g.allPlayersHaveOffice():
		err = sn.NewVError("You must first assign all players an office")
	}
	return
}
