package tammany

import (
	"bytes"
	"encoding/gob"
	"html/template"

	"github.com/gin-gonic/gin"

	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.RegisterName("*game.awardChipsEntry", new(awardChipsEntry))
}

func (g *Game) startAwardChipsPhase(ctx context.Context) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Entering")

	g.Phase = awardFavorChips
	g.awardChips()
}

func (g *Game) awardChips() {
	nationalities := g.Nationalities()
	awardChips := make(map[int]Chips, len(g.Players()))

	for _, player := range g.Players() {
		awardChips[player.ID()] = Chips{}
	}

	for _, nationality := range nationalities {
		winners := g.awardChipsFor(nationality)
		for _, winner := range winners {
			awardChips[winner.ID()][nationality] = 3
		}
	}
	e := g.newAwardChipsEntry()
	e.ChipWinners = awardChips
}

type awardChipsEntry struct {
	*Entry
	ChipWinners map[int]Chips
}

func (g *Game) newAwardChipsEntry() (e *awardChipsEntry) {
	e = new(awardChipsEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return
}

func (e *awardChipsEntry) HTML(ctx context.Context) template.HTML {
	g := gameFrom(ctx)
	ts := restful.TemplatesFrom(ctx)
	buf := new(bytes.Buffer)
	tmpl := ts["tammany/award_chips_entry"]
	if err := tmpl.Execute(buf, gin.H{
		"entry": e,
		"g":     g,
		"ctx":   ctx,
	}); err != nil {
		return ""
	}
	return restful.HTML(buf.String())
}

func (g *Game) awardChipsFor(n nationality) (winners Players) {
	winners = g.chipWinners(n)
	for _, player := range winners {
		player.Chips[n] += 3
	}
	return
}

func (g *Game) chipWinners(n nationality) (winners Players) {
	var max int
	winners = make(Players, 0)
	for _, player := range g.Players() {
		controlled := g.ControlledBy(player, n)
		switch {
		case controlled > max:
			max = controlled
			winners = Players{player}
		case controlled == max:
			winners = append(winners, player)
		}
	}
	return
}
