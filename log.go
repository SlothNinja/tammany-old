package tammany

import (
	"fmt"
	"html/template"
	"time"

	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"golang.org/x/net/context"
)

// Entry provides a game entry for the game log.
type Entry struct {
	*game.Entry
}

// GameLog stores entries of the game log.
type GameLog []Entryer

// Entryer specifies the interface for entries of the game log.
type Entryer interface {
	PhaseName() string
	Turn() int
	Round() int
	CreatedAt() time.Time
	HTML(context.Context) template.HTML
}

func (g *Game) newEntry() (e *Entry) {
	e = new(Entry)
	e.Entry = new(game.Entry)
	e.PlayerID = game.NoPlayerID
	e.OtherPlayerID = game.NoPlayerID
	e.TurnF = g.Turn
	e.PhaseF = g.Phase
	e.SubPhaseF = g.SubPhase
	e.RoundF = g.Round
	e.CreatedAtF = time.Now()
	return
}

func (g *Game) newEntryFor(p *Player) (e *Entry) {
	e = g.newEntry()
	e.PlayerID = p.ID()
	return
}

// PhaseName displays the turn and phase in an entry of the game log.
func (e *Entry) PhaseName() string {
	return fmt.Sprintf("Year %d | Phase: %s", e.Round(), phaseNames[e.Phase()])
}

func (gl GameLog) Last() (e Entryer) {
	if l := len(gl); l > 0 {
		e = gl[l-1]
	}
	return
}
